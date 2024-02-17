package eio_test

import (
	"context"
	"fmt"
	"github.com/qmstar0/eio"
	"github.com/qmstar0/eio/message"
	"testing"
	"time"
)

var (
	TimeOut = time.Second * 5
)

func producer(ctx context.Context, topic string, pub eio.Publisher) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := pub.Publish(topic, message.WithPayload(eio.NewUUID(), []byte("hi")))
			if err != nil {
				fmt.Println("err", err)
				return
			}
			time.Sleep(time.Millisecond * 200)
		}
	}
}

func TestHandler_Middleware(t *testing.T) {
	ctx, cancle := context.WithTimeout(context.Background(), TimeOut)
	defer cancle()
	pubsub := eio.NewGoPubsub("pubsub", eio.GoPubsubConfig{})

	go producer(ctx, "main", pubsub)

	handlerMain := eio.NewHandler("main", pubsub, func(msgCtx *message.Context) error {
		t.Log("main", msgCtx, msgCtx.Err())
		return nil
	})

	handlerMain.AddMiddleware(func(fn eio.HandlerFunc) eio.HandlerFunc {
		return func(msg *message.Context) error {
			t.Log("main执行前-m1")
			err := fn(msg)
			if err != nil {
				return err
			}
			t.Log("main执行后-m1")
			return nil
		}
	})

	t.Log(handlerMain)

	handlerMain.Run(ctx, func(fn eio.HandlerFunc) eio.HandlerFunc {
		return func(msg *message.Context) error {
			t.Log("main执行前-m2")
			err := fn(msg)
			if err != nil {
				return err
			}
			t.Log("main执行后-m2")
			return nil
		}
	})
}
func TestHandler_Stop(t *testing.T) {
	ctx, cancle := context.WithTimeout(context.Background(), TimeOut)
	defer cancle()
	pubsub := eio.NewGoPubsub("pubsub", eio.GoPubsubConfig{})

	go producer(ctx, "main", pubsub)
	handlerMain := eio.NewHandler("main", pubsub, func(msgCtx *message.Context) error {
		t.Log("main", msgCtx)
		return nil
	})

	go handlerMain.Run(ctx)

	time.Sleep(time.Second)

	handlerMain.Stop()
}
func TestNewHandler(t *testing.T) {
	ctx, cancle := context.WithTimeout(context.Background(), TimeOut)
	defer cancle()
	pubsub := eio.NewGoPubsub("pubsub", eio.GoPubsubConfig{})
	go producer(ctx, "main", pubsub)

	handlerMain := eio.NewHandler("main", pubsub, func(msgCtx *message.Context) error {
		t.Log("main", msgCtx)
		return nil
	})
	handlerMain.Run(ctx)
}
