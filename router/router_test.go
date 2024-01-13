package router_test

import (
	"EventDriven/message"
	"EventDriven/pubsub"
	"EventDriven/pubsub/gopubsub"
	"EventDriven/router"
	"context"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/google/uuid"
	"testing"
	"time"
)

func producer(ctx context.Context, topic string, pub pubsub.Publisher) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := pub.Publish(topic, message.NewMessage(uuid.New().String(), []byte("hi")))
			if err != nil {
				fmt.Println("err", err)
				return
			}
			time.Sleep(time.Second)
		}
	}
}

func TestNewRouter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	goPubsub := gopubsub.NewGoPubsub("test", EventBus.New())

	newRouter, err := router.NewRouter(router.RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	newRouter.AddHandler(
		"test1",
		"S_T",
		goPubsub,
		func(msg *message.Message) ([]*message.Message, error) {
			t.Log("topic:S_T | handle执行")
			return nil, nil
		})

	go producer(ctx, "S_T", goPubsub)

	err = newRouter.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

}

func TestHandler_Dispatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	goPubsub := gopubsub.NewGoPubsub("test", EventBus.New())

	newRouter, err := router.NewRouter(router.RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	handler1 := newRouter.AddHandler(
		"test1",
		"S_T",
		goPubsub,
		func(msg *message.Message) ([]*message.Message, error) {
			t.Log("topic:S_T | handle执行")
			return []*message.Message{msg}, nil
		})

	newRouter.AddHandler(
		"test2",
		"P_T",
		goPubsub,
		func(msg *message.Message) ([]*message.Message, error) {
			t.Log("topic:P_T | handle执行")
			return nil, nil
		})

	handler1.Dispatch("P_T", goPubsub)

	go producer(ctx, "S_T", goPubsub)

	err = newRouter.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRouter_AddHandleMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	goPubsub := gopubsub.NewGoPubsub("test", EventBus.New())

	newRouter, err := router.NewRouter(router.RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	newRouter.AddHandler(
		"test1",
		"S_T",
		goPubsub,
		func(msg *message.Message) ([]*message.Message, error) {
			t.Log("topic:S_T | handle执行")
			return []*message.Message{msg}, nil
		})
	newRouter.AddHandleMiddleware(
		func(fn router.DispatchHandleFunc) router.DispatchHandleFunc {
			return func(msg *message.Message) ([]*message.Message, error) {
				t.Log("1号中间件，在base handler运行前")
				messages, err := fn(msg)
				if err != nil {
					return messages, err
				}
				t.Log("1号中间件，在base handler运行后", messages)
				return messages, err
			}
		},
		func(fn router.DispatchHandleFunc) router.DispatchHandleFunc {
			return func(msg *message.Message) ([]*message.Message, error) {
				t.Log("2号中间件，在base handler运行前")
				messages, err := fn(msg)
				if err != nil {
					return messages, err
				}
				t.Log("2号中间件，在base handler运行后")
				return messages, err
			}
		},
	)

	go producer(ctx, "S_T", goPubsub)

	err = newRouter.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

}

func TestHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	goPubsub := gopubsub.NewGoPubsub("test", EventBus.New())

	newRouter, err := router.NewRouter(router.RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}
	handler := newRouter.AddHandler(
		"test1",
		"S_T",
		goPubsub,
		func(msg *message.Message) ([]*message.Message, error) {
			t.Log("topic:S_T | handle执行")
			return []*message.Message{msg}, nil
		})
	newRouter.AddHandleMiddleware(
		func(fn router.DispatchHandleFunc) router.DispatchHandleFunc {
			return func(msg *message.Message) ([]*message.Message, error) {
				t.Log("router中间件，在basehandler运行前")
				messages, err := fn(msg)
				if err != nil {
					return messages, err
				}
				t.Log("router中间件，在basehandler运行后")
				return messages, err
			}
		})

	handler.AddHandleMiddleware(func(fn router.DispatchHandleFunc) router.DispatchHandleFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			t.Log("handler中间件，在basehandler运行前")
			messages, err := fn(msg)
			if err != nil {
				return messages, err
			}
			t.Log("handler中间件，在basehandler运行后")
			return messages, err
		}
	})
	handler.Dispatch("P_T", goPubsub)

	newRouter.AddHandler(
		"test2",
		"P_T",
		goPubsub,
		func(msg *message.Message) ([]*message.Message, error) {
			t.Log("topic:P_T | handle执行")
			return nil, nil
		},
	)

	go producer(ctx, "S_T", goPubsub)

	err = newRouter.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
