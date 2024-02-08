package processor_test

import (
	"context"
	"github.com/qmstar0/eio"
	"github.com/qmstar0/eio/message"
	"github.com/qmstar0/eio/processor"
	"github.com/qmstar0/eio/pubsub/gopubsub"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func createPubSub() (eio.Publisher, eio.Subscriber) {
	pubsub := gopubsub.NewGoPubsub("test", gopubsub.GoPubsubConfig{})
	return pubsub, pubsub
}

func createRouter() processor.Router {
	return processor.NewRouter()
}

func publishMessage(t *testing.T, ctx context.Context, topic string, publisher eio.Publisher) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := publisher.Publish(topic, message.WithPayload(eio.NewUUID(), []byte("hi")))
			assert.NoError(t, err)
			time.Sleep(time.Millisecond * 200)
		}
	}
}

func Test(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	pub, sub := createPubSub()
	router := createRouter()

	handler := router.AddHandler("1", "main", sub, func(msg *message.Context) ([]*message.Context, error) {
		t.Log("handler-main", msg)
		return []*message.Context{msg}, nil
	})

	router.AddHandler("2", "sub", sub, func(msg *message.Context) ([]*message.Context, error) {
		t.Log("handler-sub", msg)
		return []*message.Context{msg}, nil
	})

	router.AddMiddleware(func(fn processor.HandlerFunc) processor.HandlerFunc {
		return func(msg *message.Context) ([]*message.Context, error) {
			t.Log("router-middleware 在handler前")
			messages, err := fn(msg)
			assert.NoError(t, err)
			assert.Equal(t, len(messages), 1)
			t.Log("router-middleware 在handler后")
			return messages, nil
		}
	})

	handler.AddMiddleware(func(fn processor.HandlerFunc) processor.HandlerFunc {
		return func(msg *message.Context) ([]*message.Context, error) {
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

	go func() {
		<-router.Running()
		assert.True(t, router.IsRunning())
		assert.False(t, router.IsClosed())

		time.Sleep(time.Second * 2)

		err := router.Close()
		assert.NoError(t, err)

		assert.False(t, router.IsRunning())
		assert.True(t, router.IsClosed())

		select {
		case <-router.Running():
			panic("关闭后的router`<-router.Running()`仍在运行")
		default:

		}

	}()

	err := router.Run(ctx)
	assert.NoError(t, err)

	t.Log("重新运行")
	time.Sleep(time.Second)

	err = router.Run(ctx)
	assert.NoError(t, err)
}

func TestRouterCloseTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pub, sub := createPubSub()
	router := processor.NewRouterWithConfig(processor.RouterConfig{CloseTimeout: time.Second * 2})

	router.AddHandler("1", "main", sub, func(msg *message.Context) ([]*message.Context, error) {
		t.Log("main", msg)
		// 每个handler阻塞5秒
		time.Sleep(time.Second * 5)
		return nil, nil
	})

	go producer(ctx, "main", pub)

	go func() {
		err := router.Run(ctx)
		assert.NoError(t, err)
	}()

	// 运行两秒后关闭router
	time.Sleep(time.Second * 2)

	err := router.Close()
	assert.Error(t, err)
}

func TestRouterRuntimeHandlerStep(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pub, sub := createPubSub()
	router := processor.NewRouterWithConfig(processor.RouterConfig{CloseTimeout: time.Second * 2})

	handler := router.AddHandler("1", "main", sub, func(msg *message.Context) ([]*message.Context, error) {
		t.Log("main", msg)
		// 每个handler阻塞3秒
		time.Sleep(time.Second * 3)
		return nil, nil
	})

	go producer(ctx, "main", pub)

	//测试当router中的handler停止后，router是否正常关闭
	go func() {

		time.Sleep(time.Second * 4)
		handler.Stop()
		t.Log("handler stoped")
	}()

	err := router.Run(ctx)
	assert.NoError(t, err)
}
