package processor_test

import (
	"context"
	"fmt"
	"github.com/qmstar0/eio"
	"github.com/qmstar0/eio/message"
	"github.com/qmstar0/eio/processor"
	"github.com/qmstar0/eio/pubsub"
	"github.com/qmstar0/eio/pubsub/gopubsub"
	"testing"
	"time"
)

var (
	TimeOut = time.Second * 5
)

func producer(ctx context.Context, topic string, pub pubsub.Publisher) {
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
	pubsub := gopubsub.NewGoPubsub("pubsub", gopubsub.GoPubsubConfig{})

	go producer(ctx, "main", pubsub)

	handlerMain := processor.NewHandler("main", pubsub, func(msgCtx *message.Context) error {
		t.Log("main", msgCtx, msgCtx.Err())
		return nil
	})

	handlerMain.AddMiddleware(func(fn processor.HandlerFunc) processor.HandlerFunc {
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

	handlerMain.Run(ctx, func(fn processor.HandlerFunc) processor.HandlerFunc {
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
	pubsub := gopubsub.NewGoPubsub("pubsub", gopubsub.GoPubsubConfig{})

	go producer(ctx, "main", pubsub)
	handlerMain := processor.NewHandler("main", pubsub, func(msgCtx *message.Context) error {
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
	pubsub := gopubsub.NewGoPubsub("pubsub", gopubsub.GoPubsubConfig{})
	go producer(ctx, "main", pubsub)

	handlerMain := processor.NewHandler("main", pubsub, func(msgCtx *message.Context) error {
		t.Log("main", msgCtx)
		return nil
	})
	handlerMain.Run(ctx)
}
