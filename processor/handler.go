package processor

import (
	"context"
	"fmt"
	"github.com/qmstar0/eio"
	"github.com/qmstar0/eio/message"
	"sync"
)

type HandlerFunc func(msg *message.Context) error

type HandlerMiddleware func(fn HandlerFunc) HandlerFunc

type Handler struct {
	subscriber      eio.Subscriber
	subscriberTopic string

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

		started:   false,
		startedCh: make(chan struct{}),
	}
}

func (h *Handler) Run(ctx context.Context, middlewares ...HandlerMiddleware) {

	allMiddlewares := append(append([]HandlerMiddleware(nil), middlewares...), h.middleware...)

	handlerFn := h.handlerFn

	for _, middleware := range allMiddlewares {
		handlerFn = middleware(handlerFn)
	}

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

func (h *Handler) handleMessage(msg *message.Context, handlerFn HandlerFunc) {
	defer h.runningHandlersWg.Done()

	defer func() {
		if recovered := recover(); recovered != nil {
			msg.Nack(recoverErr{recovered})
		}
	}()

	err := handlerFn(msg)
	//if err != nil {
	//	msg.Nack(err)
	//	return
	//}
	//
	//msg.Ack()
	msg.Nack(err)
}

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

type recoverErr struct {
	message any
}

func (r recoverErr) Error() string {
	return fmt.Sprintf("recovered from panic:%s", r.message)
}
