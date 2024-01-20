package processor_test

import (
	"context"
	"github.com/qmstar0/eventDriven"
	"github.com/qmstar0/eventDriven/message"
	"github.com/qmstar0/eventDriven/processor"
	"github.com/qmstar0/eventDriven/pubsub/gopubsub"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func createPubSub() (eventDriven.Publisher, eventDriven.Subscriber) {
	pubsub := gopubsub.NewGoPubsub("test", gopubsub.GoPubsubConfig{})
	return pubsub, pubsub
}

func createRouter() *processor.Router {
	return processor.NewRouter()
}

func publishMessage(t *testing.T, ctx context.Context, topic string, publisher eventDriven.Publisher) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := publisher.Publish(topic, message.NewMessage(eventDriven.NewUUID(), []byte("hi")))
			assert.NoError(t, err)
			time.Sleep(time.Millisecond * 200)
		}
	}
}

func Test(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), TimeOut)
	defer cancel()

	pub, sub := createPubSub()
	router := createRouter()

	handler := router.AddHandler("1", "main", sub, func(msg *message.Message) ([]*message.Message, error) {
		t.Log("handler-main", msg)
		return []*message.Message{msg}, nil
	})

	router.AddHandler("2", "sub", sub, func(msg *message.Message) ([]*message.Message, error) {
		t.Log("handler-sub", msg)
		return []*message.Message{msg}, nil
	})

	router.AddMiddleware(func(fn processor.HandlerFunc) processor.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			t.Log("router-middleware 在handler前")
			messages, err := fn(msg)
			assert.NoError(t, err)
			assert.Equal(t, len(messages), 1)
			t.Log("router-middleware 在handler后")
			return messages, nil
		}
	})

	handler.AddMiddleware(func(fn processor.HandlerFunc) processor.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			t.Log("handler-middleware 在handler前", msg)
			messages, err := fn(msg)
			assert.NoError(t, err)
			assert.Equal(t, len(messages), 1)
			t.Log("handler-middleware 在handler后", msg)
			return messages, nil
		}
	})

	handler.AddForword(processor.Forward("sub", pub))

	go publishMessage(t, ctx, "main", pub)

	reRun := make(chan struct{})

	go func() {
		<-router.Running()
		assert.True(t, router.IsRunning())
		assert.False(t, router.IsClosed())

		time.Sleep(time.Second * 2)

		router.Close()

		assert.False(t, router.IsRunning())
		assert.True(t, router.IsClosed())

		select {
		case <-router.Running():
			panic("关闭后的router`<-router.Running()`仍在运行")
		default:
			close(reRun)
		}

	}()

	err := router.Run(ctx)
	assert.NoError(t, err)
	<-reRun
	t.Log("重新运行")
	time.Sleep(time.Second)
	err = router.Run(ctx)
	assert.NoError(t, err)
}
