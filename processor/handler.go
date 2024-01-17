package processor

import (
	"context"
	"github.com/qmstar0/eventDriven"
	"github.com/qmstar0/eventDriven/message"
	"sync"
)

type HandlerFunc func(msg *message.Message) ([]*message.Message, error)

type HandleMiddleware func(fn HandlerFunc) HandlerFunc

type Handler struct {
	subscriber      eventDriven.Subscriber
	subscriberTopic string

	publisherMap     map[string]eventDriven.Publisher
	publisherMapLock sync.Mutex

	runningHandlersWgLock sync.Mutex
	runningHandlersWg     sync.WaitGroup

	middleware     []HandleMiddleware
	middlewareLock sync.Mutex

	handlerFn HandlerFunc

	started   bool
	startedCh chan struct{}
	stopped   chan struct{}
	stopFn    context.CancelFunc
}

func NewHandler(topic string, sub eventDriven.Subscriber, fn HandlerFunc) *Handler {
	return &Handler{
		subscriber:      sub,
		subscriberTopic: topic,

		handlerFn: fn,

		runningHandlersWg:     sync.WaitGroup{},
		runningHandlersWgLock: sync.Mutex{},

		middleware:     make([]HandleMiddleware, 0),
		middlewareLock: sync.Mutex{},

		publisherMap:     make(map[string]eventDriven.Publisher),
		publisherMapLock: sync.Mutex{},

		started:   false,
		startedCh: make(chan struct{}),
	}
}
func (h *Handler) Run(ctx context.Context) {

	//添加中间件
	for _, middleware := range h.middleware {
		handlerFn := h.handlerFn
		h.handlerFn = middleware(handlerFn)
	}

	//添加发布者
	for topic, publisher := range h.publisherMap {
		handlerFn := h.handlerFn
		h.handlerFn = h.getDecoratedFunc(handlerFn, topic, publisher)
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

		go h.handleMessage(msg, h.handlerFn)
	}

	h.startedCh = make(chan struct{})
	h.started = false
	close(h.stopped)
}

func (h *Handler) handleMessage(msg *message.Message, handlerFn HandlerFunc) {
	defer h.runningHandlersWg.Done()

	defer func() {
		if recovered := recover(); recovered != nil {
			msg.Nack()
		}
	}()

	//producedMessages, err := handler(msg)
	_, err := handlerFn(msg)
	if err != nil {
		msg.Nack()
		return
	}

	msg.Ack()
}

func (h *Handler) handleClose(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-h.stopped:
	}
	h.stopFn()
}

func (h *Handler) getDecoratedFunc(handlerFn HandlerFunc, topic string, pub eventDriven.Publisher) HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		var err error
		msgs, err := handlerFn(msg)
		if err != nil {
			return msgs, err
		}
		for _, m := range msgs {
			err = pub.Publish(topic, m)
			if err != nil {
				return msgs, err
			}
		}
		return msgs, nil
	}
}

func (h *Handler) Use(ms ...HandleMiddleware) {
	h.middlewareLock.Lock()
	defer h.middlewareLock.Unlock()
	h.middleware = append(h.middleware, ms...)
}

func (h *Handler) Repost(trpostHandler *Handler, pub eventDriven.Publisher) {
	h.publisherMapLock.Lock()
	defer h.publisherMapLock.Unlock()
	h.publisherMap[trpostHandler.subscriberTopic] = pub
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
