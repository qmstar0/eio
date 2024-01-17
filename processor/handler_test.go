package processor_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/qmstar0/eventDriven"
	"github.com/qmstar0/eventDriven/message"
	"github.com/qmstar0/eventDriven/processor"
	"github.com/qmstar0/eventDriven/pubsub/gopubsub"
	"testing"
	"time"
)

var (
	TimeOut = time.Second * 3
)

func producer(ctx context.Context, topic string, pub eventDriven.Publisher) {
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
			time.Sleep(time.Millisecond * 200)
		}
	}
}

func TestNewHandler(t *testing.T) {

	pubsub := gopubsub.NewGoPubsub("pubsub", gopubsub.GoPubsubConfig{})

	handlerMain := processor.NewHandler("main", pubsub, func(msg *message.Message) ([]*message.Message, error) {
		t.Log("main", msg)
		return []*message.Message{msg}, nil
	})

	t.Run("test repost", func(t *testing.T) {
		ctx, cancle := context.WithTimeout(context.Background(), TimeOut)
		defer cancle()

		go producer(ctx, "main", pubsub)

		handlerSub := processor.NewHandler("sub", pubsub, func(msg *message.Message) ([]*message.Message, error) {
			t.Log("sub", msg)
			return []*message.Message{msg}, nil
		})
		handlerMain.Repost(handlerSub, pubsub)
		go handlerSub.Run(ctx)
		handlerMain.Run(ctx)
	})

	//test middleware
	t.Run("test middleware", func(t *testing.T) {
		ctx, cancle := context.WithTimeout(context.Background(), TimeOut)
		defer cancle()

		go producer(ctx, "main", pubsub)

		handlerMain.Use(func(fn processor.HandlerFunc) processor.HandlerFunc {
			return func(msg *message.Message) ([]*message.Message, error) {
				t.Log("main执行前")
				messages, err := fn(msg)
				if err != nil {
					return messages, err
				}
				t.Log("main执行后", messages)
				return messages, nil
			}
		})
		handlerMain.Run(ctx)
	})

	t.Run("test stop", func(t *testing.T) {
		ctx, cancle := context.WithTimeout(context.Background(), TimeOut)
		defer cancle()

		go producer(ctx, "main", pubsub)

		go handlerMain.Run(ctx)

		time.Sleep(time.Second)

		handlerMain.Stop()
	})
}

//func testHandler_Use(t *testing.T, pubsub *gopubsub.GoPubsub) {
//
//}
//
//func testHandler_Repost(t *testing.T, pubsub *gopubsub.GoPubsub) {
//}
//
//func testHandler_Run(t *testing.T, pubsub *gopubsub.GoPubsub) {
//}
//
//func testHandler_Stop(t *testing.T, pubsub *gopubsub.GoPubsub) {
//}
