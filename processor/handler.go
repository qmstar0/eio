package processor

import (
	"context"
	"github.com/qmstar0/eio"
	"github.com/qmstar0/eio/message"
	"sync"
)

type HandlerFunc func(msg *message.Message) ([]*message.Message, error)

type HandlerMiddleware func(fn HandlerFunc) HandlerFunc

type Handler struct {
	subscriber      eio.Subscriber
	subscriberTopic string

	forwarders     []Forwarder
	forwardersLock *sync.Mutex

	//publisherMap     map[string]eio.publisher
	//publisherMapLock sync.Mutex

	runningHandlersWgLock *sync.Mutex
	runningHandlersWg     *sync.WaitGroup

	middleware     []HandlerMiddleware
	middlewareLock *sync.Mutex

	handlerFn HandlerFunc

	started   bool
	startedCh chan struct{}
	stopped   chan struct{}
	stopFn    context.CancelFunc
}

func NewHandler(topic string, sub eio.Subscriber, fn HandlerFunc) *Handler {
	return &Handler{
		subscriber:      sub,
		subscriberTopic: topic,

		handlerFn: fn,

		runningHandlersWg:     &sync.WaitGroup{},
		runningHandlersWgLock: &sync.Mutex{},

		middleware:     make([]HandlerMiddleware, 0),
		middlewareLock: &sync.Mutex{},

		forwarders:     make([]Forwarder, 0),
		forwardersLock: &sync.Mutex{},

		//publisherMap:     make(map[string]eio.publisher),
		//publisherMapLock: sync.Mutex{},

		started:   false,
		startedCh: make(chan struct{}),
	}
}

func (h *Handler) Run(ctx context.Context, middlewares ...HandlerMiddleware) {

	// 初始化一个空的[]HandlerMiddleware
	// 然后依次加入
	// - 中间件(middlewares)
	// - 以中间件形式使用的转发中间件(getForwarderMiddlewares(h.forwarders))

	allMiddlewares := append(append(append([]HandlerMiddleware(nil),
		middlewares...), h.middleware...), getForwarderMiddlewares(h.forwarders)...)

	handlerFn := h.handlerFn

	//添加中间件
	for _, middleware := range allMiddlewares {
		handlerFn = middleware(handlerFn)
	}

	//添加发布者
	//for _, forwarder := range h.forwarders {
	//	handlerFn := h.handlerFn
	//	h.handlerFn = h.getDecoratedFunc(handlerFn, forwarder.topic, forwarder.publisher)
	//}

	subCtx, cancel := context.WithCancel(ctx)

	h.started = true
	h.stopFn = cancel
	h.stopped = make(chan struct{})

	close(h.startedCh)

	go h.handleClose(ctx)

	messageCh, err := h.subscriber.Subscribe(subCtx, h.subscriberTopic)
	if err != nil {
		panic(err)
	}

	for msg := range messageCh {
		h.runningHandlersWgLock.Lock()
		h.runningHandlersWg.Add(1)
		h.runningHandlersWgLock.Unlock()

		go h.handleMessage(msg, handlerFn)
	}

	h.startedCh = make(chan struct{})
	h.started = false
	close(h.stopped)

	h.runningHandlersWg.Wait()
}

func (h *Handler) handleMessage(msg *message.Message, handlerFn HandlerFunc) {
	defer h.runningHandlersWg.Done()

	defer func() {
		if recovered := recover(); recovered != nil {
			//msg.Nack()
		}
	}()

	//producedMessages, err := handlerFn(msg)
	_, err := handlerFn(msg)
	if err != nil {
		//msg.Nack()
		return
	}

	//if len(producedMessages) != 0 {
	//	h.forwardMessage(producedMessages)
	//}

	//msg.Ack()
}

// 关于转发handler处理返回的[]*Message，有两种方法
// 1.将返回的producedMessages循环发布(forwardMessage)
// 2.以中间件的形式发布(Forward.Middleware)
// 两者在实现的复杂度上差不多，但在维护的复杂度上，我认为只使用中间件这一种形式更利于阅读和扩展
//
//func (h *Handler) forwardMessage(msgs []*message.Message) {
//	for _, forwarder := range h.forwarders {
//		err := forwarder.publisher.Publish(forwarder.topic, msgs...)
//		if err != nil {
//			return
//		}
//	}
//}

func (h *Handler) handleClose(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-h.stopped:
	}
	h.stopFn()
}

func (h *Handler) AddMiddleware(ms ...HandlerMiddleware) {
	h.middlewareLock.Lock()
	defer h.middlewareLock.Unlock()
	h.middleware = append(h.middleware, ms...)
}

func (h *Handler) AddForword(forwarder ...Forwarder) {
	h.forwardersLock.Lock()
	defer h.forwardersLock.Unlock()
	h.forwarders = append(h.forwarders, forwarder...)
}

func (h *Handler) Stop() {
	if !h.started {
		panic("handler is not started")
	}
	h.stopFn()
}

func (h *Handler) Stopped() chan struct{} {
	return h.stopped
}

func (h *Handler) Started() chan struct{} {
	return h.startedCh
}
