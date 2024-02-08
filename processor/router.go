package processor

import (
	"context"
	"errors"
	"github.com/qmstar0/eio"
	"sync"
	"time"
)

type Route interface {
	AddMiddleware(ms ...HandlerMiddleware)
	AddForword(forwarder ...Forwarder)
	Stop()
	Stopped() chan struct{}
	Started() chan struct{}
}

type Router interface {
	Run(c context.Context) error
	AddHandler(name, topic string, sub eio.Subscriber, handlerFn HandlerFunc) Route
	AddMiddleware(middlewares ...HandlerMiddleware)
	Running() <-chan struct{}
	IsRunning() bool
	IsClosed() bool
	Close() error
}

type RouterConfig struct {
	CloseTimeout time.Duration
	// 也许以后有扩展
}

func (c *RouterConfig) setDefault() {
	if c.CloseTimeout <= 0 {
		c.CloseTimeout = time.Second * 10
	}
}

type router struct {
	config RouterConfig

	handlers          map[string]*Handler
	handlersLock      *sync.RWMutex
	runninghandlersWg *sync.WaitGroup

	middleware     []HandlerMiddleware
	middlewareLock *sync.Mutex

	leastOneHandlerRunning chan struct{}

	isRunning bool
	runningCh chan struct{}

	closingCh  chan struct{}
	closedCh   chan struct{}
	closedLock sync.Mutex
	closed     bool
}

func NewRouterWithConfig(config RouterConfig) Router {
	config.setDefault()
	r := &router{
		config: config,

		handlers:          make(map[string]*Handler),
		handlersLock:      &sync.RWMutex{},
		runninghandlersWg: &sync.WaitGroup{},

		middleware:     make([]HandlerMiddleware, 0),
		middlewareLock: &sync.Mutex{},

		leastOneHandlerRunning: make(chan struct{}),

		isRunning: false,
		runningCh: make(chan struct{}),

		closingCh:  make(chan struct{}),
		closedCh:   make(chan struct{}),
		closedLock: sync.Mutex{},
		closed:     false,
	}
	r.resetCloseState()
	return r
}

func NewRouter() Router {
	return NewRouterWithConfig(RouterConfig{})
}

func (r *router) Run(c context.Context) error {
	r.resetCloseState()

	if r.isRunning {
		return errors.New("router is already running")
	}
	r.isRunning = true

	routerCtx, cancel := context.WithCancel(c)
	defer cancel()

	if err := r.runHandlers(routerCtx); err != nil {
		return err
	}

	close(r.runningCh)

	go r.closeWhenAllHandlersStopped(routerCtx)

	<-r.closingCh
	cancel()

	<-r.closedCh

	return nil

}

func (r *router) runHandlers(routerCtx context.Context) error {

	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	for name, h := range r.handlers {

		if h.started {
			continue
		}

		handlerCtx, cancel := context.WithCancel(routerCtx)

		go func(n string, handler *Handler) {
			defer cancel()

			r.runninghandlersWg.Add(1)
			defer r.runninghandlersWg.Done()

			select {
			case r.leastOneHandlerRunning <- struct{}{}:
			default:
			}

			handler.Run(handlerCtx, r.middleware...)

		}(name, h)
	}
	return nil
}

func (r *router) AddHandler(name, topic string, sub eio.Subscriber, handlerFn HandlerFunc) Route {
	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	if _, ok := r.handlers[name]; ok {
		panic("不可重复添加handler")
	}

	handler := NewHandler(topic, sub, handlerFn)

	r.handlers[name] = handler

	return handler
}

func (r *router) AddMiddleware(middlewares ...HandlerMiddleware) {
	r.middlewareLock.Lock()
	defer r.middlewareLock.Unlock()
	r.middleware = append(r.middleware, middlewares...)
}

func (r *router) closeWhenAllHandlersStopped(ctx context.Context) {
	r.handlersLock.RLock()
	hasHandlers := len(r.handlers) != 0
	r.handlersLock.RUnlock()

	if hasHandlers {
		select {
		case <-r.leastOneHandlerRunning:
		case <-r.closedCh:
			return
		}
	}

	r.runninghandlersWg.Wait()
	select {
	case <-ctx.Done():
	default:
	}

	err := r.Close()
	if err != nil {

	}
}

func (r *router) waitForHandlersTimeouted() bool {
	signal := make(chan struct{})
	go func() {
		r.runninghandlersWg.Wait()
		close(signal)
	}()

	select {
	case <-signal:
		return false
	case <-time.After(r.config.CloseTimeout):
		return true
	}
}

func (r *router) Running() <-chan struct{} {
	return r.runningCh
}

func (r *router) IsRunning() bool {
	select {
	case <-r.runningCh:
		return true
	default:
		return false
	}
}

func (r *router) IsClosed() bool {
	r.closedLock.Lock()
	defer r.closedLock.Unlock()

	return r.closed
}

func (r *router) Close() error {
	defer r.resetRunningState()

	r.closedLock.Lock()
	defer r.closedLock.Unlock()

	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true
	close(r.closingCh)
	defer close(r.closedCh)

	timeouted := r.waitForHandlersTimeouted()
	if timeouted {
		return errors.New("router close timeout")
	}
	return nil
}

func (r *router) resetRunningState() {
	r.runningCh = make(chan struct{})
	r.isRunning = false
}

func (r *router) resetCloseState() {
	r.closingCh = make(chan struct{})
	r.closedCh = make(chan struct{})
	r.closed = false
}
