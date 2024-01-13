package router

import (
	"EventDriven/pubsub"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RouterConfig Router配置
type RouterConfig struct {
	// 关闭Router时的超时时长
	CloseTimeout time.Duration
}

// setDefaults 设置默认值
func (c *RouterConfig) setDefaults() {
	if c.CloseTimeout == 0 {
		c.CloseTimeout = time.Second * 30
	}
}

// Router 根据提供的handlerFunc组织handler处理来自Subscriber的消息
//
// Router只负责构建handler，组织handler的run、stop和close等策略
type Router struct {
	config RouterConfig

	handlers     map[string]*handler
	handlersLock *sync.RWMutex

	handlersWg *sync.WaitGroup

	runningHandlersWg     *sync.WaitGroup
	runningHandlersWgLock *sync.Mutex

	handleMiddleware     []HandleMiddleware
	handleMiddlewareLock *sync.Mutex

	handlerAdded chan struct{}

	isCloseingCh chan struct{}
	closedCh     chan struct{}
	closed       bool
	closedLock   sync.Mutex

	isRunning bool
	running   chan struct{}
}

func NewRouter(config RouterConfig) (*Router, error) {
	config.setDefaults()

	return &Router{
		config: config,

		handlers:     map[string]*handler{},
		handlersLock: &sync.RWMutex{},

		handlersWg: &sync.WaitGroup{},

		runningHandlersWg:     &sync.WaitGroup{},
		runningHandlersWgLock: &sync.Mutex{},

		handleMiddleware:     make([]HandleMiddleware, 0),
		handleMiddlewareLock: &sync.Mutex{},

		handlerAdded: make(chan struct{}),

		isCloseingCh: make(chan struct{}),
		closedCh:     make(chan struct{}),

		running: make(chan struct{}),
	}, nil
}

// Run 运行Router
// ctx 外部ctx
// 当ctx.Done()，Router将停止运行
func (r *Router) Run(allHandlerCtx context.Context) (err error) {
	if r.isRunning {
		return errors.New("router is already running")
	}
	r.isRunning = true

	// allHandlerCtx Router级ctx
	// 控制所有handler
	allHandlerCtx, cancel := context.WithCancel(allHandlerCtx)
	defer cancel()

	if err := r.RunHandlers(allHandlerCtx); err != nil {
		return err
	}

	// 设置状态 running
	close(r.running)

	// 启动close进程
	// 当所有handler停止时关闭Router
	go r.closeWhenAllHandlersStopped(allHandlerCtx)

	// 等待开始Close
	<-r.isCloseingCh

	// 关闭所有handler
	cancel()

	// 等待关闭信号
	<-r.closedCh

	return nil
}

func (r *Router) RunHandlers(ctx context.Context) error {
	if !r.isRunning {
		return errors.New("you can't call RunHandlers on non-running router")
	}

	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	for name, h := range r.handlers {
		name := name
		h := h

		if h.started {
			continue
		}

		// 初始化并运行handler
		//
		// handlerCtx handler级ctx
		// handlerCtx 控制当前handler，控制当前handler订阅的message通道
		handlerCtx, cancel := context.WithCancel(ctx)

		messages, err := h.subscriber.Subscribe(handlerCtx, h.subscribeTopic)
		if err != nil {
			cancel()
			return fmt.Errorf("cannot subscribe topic %s", h.subscribeTopic)
		}

		h.messagesCh = messages
		h.started = true
		close(h.startedCh)

		// handler停止方法
		h.stopFn = cancel
		h.stopped = make(chan struct{})

		go func() {
			defer cancel()

			// 启动handler
			h.run(handlerCtx, r.handleMiddleware)

			r.handlersWg.Done()

			r.handlersLock.Lock()
			delete(r.handlers, name)
			r.handlersLock.Unlock()
		}()
	}

	return nil
}

func (r *Router) AddHandleMiddleware(fns ...HandleMiddleware) {
	r.handleMiddlewareLock.Lock()
	defer r.handleMiddlewareLock.Unlock()
	for _, fn := range fns {
		r.handleMiddleware = append(r.handleMiddleware, fn)
	}
}

func (r *Router) AddHandler(
	handlerName string,
	subscribeTopic string,
	subscriber pubsub.Subscriber,
	handlerFn DispatchHandleFunc,
) *Handler {
	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	if _, ok := r.handlers[handlerName]; ok {
		panic(AddHandlerRepeatErr{HandlerName: handlerName})
	}

	newHandler := &handler{
		name:           handlerName,
		subscribeTopic: subscribeTopic,
		subscriber:     subscriber,

		publisherMap:     make(map[string]pubsub.Publisher),
		publisherMapLock: &sync.Mutex{},

		handlerFn: handlerFn,

		runningHandlersWg:     r.runningHandlersWg,
		runningHandlersWgLock: r.runningHandlersWgLock,

		handleMiddleware:     make([]HandleMiddleware, 0),
		handleMiddlewareLock: &sync.Mutex{},

		messagesCh: nil,

		startedCh: make(chan struct{}),

		routersCloseCh: r.isCloseingCh,
	}

	r.handlersWg.Add(1)
	r.handlers[handlerName] = newHandler

	// 确保至少有一个handler存在
	select {
	case r.handlerAdded <- struct{}{}:
	default:
	}

	return &Handler{
		handler: newHandler,
	}
}

func (r *Router) closeWhenAllHandlersStopped(ctx context.Context) {
	r.handlersLock.RLock()
	notHasHandlers := len(r.handlers) == 0
	r.handlersLock.RUnlock()

	//当没有handler时，执行下下面一段逻辑
	//当没有handler时，意味着用户可能在运行时添加handler，所以要等待handler至少添加一次后，在执行Router的关闭逻辑
	if notHasHandlers {
		select {
		case <-r.handlerAdded:
		case <-r.closedCh:
			// let's avoid goroutine leak
			return
		}
	}

	r.handlersWg.Wait()
	if r.IsClosed() {
		// already closed
		return
	}

	select {
	case <-ctx.Done():
	default:
	}

	if err := r.Close(); err != nil {
		return
	}
}

// Close 关闭路由
func (r *Router) Close() error {
	r.closedLock.Lock()
	defer r.closedLock.Unlock()

	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	close(r.isCloseingCh)
	defer close(r.closedCh)

	timeouted := r.waitForHandlers()
	if timeouted {
		return errors.New("router close timeout")
	}

	return nil
}

// 等待所有handler
//
// 等待handler停止、等待正在处理message的handler处理结束
// 如果超时则立即返回
func (r *Router) waitForHandlers() bool {
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		r.handlersWg.Wait()
	}()
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		r.runningHandlersWgLock.Lock()
		defer r.runningHandlersWgLock.Unlock()

		r.runningHandlersWg.Wait()
	}()
	return WaitGroupTimeout(&waitGroup, r.config.CloseTimeout)
}

// IsClosed Router状态
func (r *Router) IsClosed() bool {
	r.closedLock.Lock()
	defer r.closedLock.Unlock()

	return r.closed
}
