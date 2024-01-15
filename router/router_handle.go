// 原文件地址:https://github.com/ThreeDotsLabs/watermill/blob/master/message/router.go
//
// 该部分有以下修改
// - 取消Log
// - 取消publisherDecorators
// - 取消subscriberDecorators
// - 修改Middleware相关
// - 重制为handler设置路由的方法
// - 重制后的Handler.Dispatch()基本上覆盖了转发器的作用

package router

import (
	"EventDriven/message"
	"EventDriven/pubsub"
	"context"
	"fmt"
	"sync"
)

// HandleFunc 一般的handle
type HandleFunc func(msg *message.Message) error

// DispatchHandleFunc 返回message的handle
type DispatchHandleFunc func(msg *message.Message) ([]*message.Message, error)

// HandleMiddleware 为handler添加中间件，以装饰器的形式
type HandleMiddleware func(fn DispatchHandleFunc) DispatchHandleFunc

type handler struct {
	//HandlerName
	name string

	//subscriber
	//该handler订阅的topic和使用的Subscriber
	subscribeTopic string
	subscriber     pubsub.Subscriber

	//publisherMap
	//handler返回的message会按照publisherMap重新发布
	publisherMap     map[string]pubsub.Publisher
	publisherMapLock *sync.Mutex

	// 处理方法
	handlerFn DispatchHandleFunc

	// 该handler正在运行中的数量
	runningHandlersWg     *sync.WaitGroup
	runningHandlersWgLock *sync.Mutex

	//handler级中间件
	handleMiddleware     []HandleMiddleware
	handleMiddlewareLock *sync.Mutex

	//数据读取Channel
	messagesCh <-chan *message.Message

	//handler状态相关
	started        bool
	startedCh      chan struct{}
	stopped        chan struct{}
	routersCloseCh <-chan struct{}
	stopFn         context.CancelFunc
}

func (h handler) run(ctx context.Context, middlewares []HandleMiddleware) {
	//添加router级中间件
	for _, middleware := range middlewares {
		handlerFn := h.handlerFn
		h.handlerFn = middleware(handlerFn)
	}

	//添加handler级中间件
	for _, middleware := range h.handleMiddleware {
		handlerFn := h.handlerFn
		h.handlerFn = middleware(handlerFn)
	}

	//添加发布者
	for topic, publisher := range h.publisherMap {
		handlerFn := h.handlerFn
		h.handlerFn = h.getDecoratedFunc(handlerFn, topic, publisher)
	}

	go h.handleClose(ctx)

	for msg := range h.messagesCh {
		h.runningHandlersWgLock.Lock()
		h.runningHandlersWg.Add(1)
		h.runningHandlersWgLock.Unlock()

		go h.handleMessage(msg, h.handlerFn)
	}

	close(h.stopped)
}

// 开启close进程
func (h handler) handleClose(ctx context.Context) {
	select {
	case <-h.routersCloseCh:
		// for backward compatibility we are closing subscriber
		if err := h.subscriber.Close(); err != nil {
		}
	case <-ctx.Done():
		// we are closing subscriber just when entire router is closed
	}
	h.stopFn()
}

// 处理message
func (h handler) handleMessage(msg *message.Message, handler DispatchHandleFunc) {
	defer h.runningHandlersWg.Done()

	defer func() {
		if recovered := recover(); recovered != nil {
			msg.Nack()
		}
	}()

	//producedMessages, err := handler(msg)
	_, err := handler(msg)
	if err != nil {
		msg.Nack()
		return
	}

	msg.Ack()
}

// 通过装饰器的形式为handler设置转发
func (h handler) getDecoratedFunc(handlerFn DispatchHandleFunc, topic string, pub pubsub.Publisher) DispatchHandleFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		var err error
		msgs, err := handlerFn(msg)
		if err != nil {
			return msgs, err
		}
		for _, m := range msgs {
			fmt.Println("getDecoratedFunc.topic", topic)
			err = pub.Publish(topic, m)
			if err != nil {
				return msgs, err
			}
		}
		return msgs, nil
	}
}

// Handler handler需要对外暴露一些方法
type Handler struct {
	handler *handler
}

func (h Handler) Dispatch(topic string, pub pubsub.Publisher) {

	//重复添加可能会导致用户错过一些东西
	h.handler.publisherMapLock.Lock()
	if _, ok := h.handler.publisherMap[topic]; ok {
		panic(AddDispatchRepeatErr{Topic: topic})
	} else {
		h.handler.publisherMap[topic] = pub
	}
	h.handler.publisherMapLock.Unlock()

}

func (h Handler) AddHandleMiddleware(fns ...HandleMiddleware) {
	h.handler.handleMiddlewareLock.Lock()
	defer h.handler.handleMiddlewareLock.Unlock()
	for _, fn := range fns {
		h.handler.handleMiddleware = append(h.handler.handleMiddleware, fn)
	}
}

// Started returns channel which is stopped when handler is running.
func (h Handler) Started() chan struct{} {
	return h.handler.startedCh
}

// Stop stops the handler.
// Stop is asynchronous.
// You can check if handler was stopped with Stopped() function.
func (h Handler) Stop() {
	if !h.handler.started {
		panic("handler is not started")
	}

	h.handler.stopFn()
}

// Stopped returns channel which is stopped when handler did stop.
func (h Handler) Stopped() chan struct{} {
	return h.handler.stopped
}
