package processor

import (
	"context"
	"errors"
	"github.com/qmstar0/eventDriven"
	"sync"
)

type RouterConfig struct {
}

type Router struct {
	config RouterConfig

	forwards     []*Forwarder
	forwardsLock sync.Mutex

	handlers     map[string]*Handler
	handlersLock *sync.RWMutex
	handlersWg   *sync.WaitGroup

	middleware     []HandlerMiddleware
	middlewareLock *sync.Mutex

	leastAHandler chan struct{}

	isRunning bool
	runningCh chan struct{}

	closingCh  chan struct{}
	closedCh   chan struct{}
	closedLock sync.Mutex
	closed     bool
}

func (r *Router) Run(c context.Context) error {
	if r.isRunning {
		return errors.New("router is already running")
	}
	r.isRunning = true

	routerCtx, cancel := context.WithCancel(c)
	defer cancel()

	if err := r.RunHandlers(routerCtx); err != nil {
		return err
	}

	close(r.runningCh)

	go r.closeWhenAllHandlersStopped(routerCtx)

	<-r.closingCh
	cancel()

	<-r.closedCh

	return nil

}

func (r *Router) RunHandlers(routerCtx context.Context) error {
	if !r.isRunning {
		return errors.New("you can't call RunHandlers on non-running router")
	}

	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	for name, h := range r.handlers {

		if h.started {
			continue
		}

		handlerCtx, cancel := context.WithCancel(routerCtx)

		go func(n string, handler *Handler) {
			defer cancel()

			// 在handler.Run期间锁定中间件和转发者，防止无效更新引发的人为逻辑错误
			handler.middlewareLock.Lock()
			defer handler.middlewareLock.Unlock()
			handler.forwardersLock.Lock()
			defer handler.forwardersLock.Unlock()

			// 备份中间件和转发者，在handler.Run结束后恢复
			middlewareBackup := append([]HandlerMiddleware(nil), handler.middleware...)
			forwardersBackup := append([]*Forwarder(nil), handler.forwarders...)
			defer func() {
				handler.middleware = middlewareBackup
				handler.forwarders = forwardersBackup
			}()

			handler.middleware = append(r.middleware, handler.middleware...)
			handler.forwarders = append(r.forwards, handler.forwarders...)

			handler.Run(handlerCtx)

			r.handlersWg.Done()

			r.handlersLock.Lock()
			delete(r.handlers, n)
			r.handlersLock.Unlock()
		}(name, h)
	}
	return nil
}

func (r *Router) AddHandler(name, topic string, sub eventDriven.Subscriber, handlerFn HandlerFunc) *Handler {
	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	if _, ok := r.handlers[name]; ok {
		panic("不可重复添加handler")
	}

	handler := NewHandler(topic, sub, handlerFn)

	r.handlersWg.Add(1)
	r.handlers[name] = handler

	select {
	case r.leastAHandler <- struct{}{}:
	default:
	}

	return handler
}

func (r *Router) AddMiddleware(middlewares ...HandlerMiddleware) {
	r.middlewareLock.Lock()
	defer r.middlewareLock.Unlock()
	r.middleware = append(r.middleware, middlewares...)
}

func (r *Router) AddForward(forwarders ...*Forwarder) {
	r.forwardsLock.Lock()
	defer r.forwardsLock.Unlock()
	r.forwards = append(r.forwards, forwarders...)
}

func (r *Router) closeWhenAllHandlersStopped(ctx context.Context) {
	r.handlersLock.RLock()
	hasHandlers := len(r.handlers) == 0
	r.handlersLock.RUnlock()

	if hasHandlers {
		select {
		case <-r.leastAHandler:
		case <-r.closedCh:
			return
		}
	}

	r.handlersWg.Wait()
	if r.IsClosed() {
		// already closed
		return
	}

	// Only log an error if the context was not canceled, but handlers were stopped.
	select {
	case <-ctx.Done():
	default:
		//r.logger.Error("All handlers stopped, closing router", errors.New("all router handlers stopped"), nil)
	}

	r.Close()
}

func (r *Router) Running() <-chan struct{} {
	return r.runningCh
}

func (r *Router) IsRunning() bool {
	select {
	case <-r.runningCh:
		return true
	default:
		return false
	}
}

func (r *Router) IsClosed() bool {
	r.closedLock.Lock()
	defer r.closedLock.Unlock()

	return r.closed
}

func (r *Router) Close() {
	r.closedLock.Lock()
	defer r.closedLock.Unlock()

	r.handlersLock.Lock()
	defer r.handlersLock.Unlock()

	if r.closed {
		return
	}
	r.closed = true

	close(r.closingCh)
	defer close(r.closedCh)

}
