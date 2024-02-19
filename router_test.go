package eio_test

import (
	"context"
	"github.com/qmstar0/eio"
	"github.com/qmstar0/eio/message"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func createRouter() eio.Router {
	return eio.NewRouter()
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pubsub := eio.NewGoPubsub(ctx, eio.GoPubsubConfig{})
	router := createRouter()

	handler := router.AddHandler("1", "main", pubsub, func(msg *message.Context) error {
		t.Log("handler-main", msg)
		return nil
	})

	router.AddHandler("2", "sub", pubsub, func(msg *message.Context) error {
		t.Log("handler-sub", msg)
		return nil
	})

	router.AddMiddleware(func(fn eio.HandlerFunc) eio.HandlerFunc {
		return func(msg *message.Context) error {
			t.Log("router-middleware 在handler前")
			err := fn(msg)
			assert.NoError(t, err)
			t.Log("router-middleware 在handler后")
			return nil
		}
	})

	handler.AddMiddleware(func(fn eio.HandlerFunc) eio.HandlerFunc {
		return func(msg *message.Context) error {
			t.Log("handler-middleware 在handler前", msg)
			err := fn(msg)
			assert.NoError(t, err)
			t.Log("handler-middleware 在handler后", msg)
			return nil
		}
	})

	go publishMessage(t, ctx, "main", pubsub)

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pubsub := eio.NewGoPubsub(ctx, eio.GoPubsubConfig{})
	router := eio.NewRouterWithConfig(eio.RouterConfig{CloseTimeout: time.Second * 2})

	router.AddHandler("1", "main", pubsub, func(msg *message.Context) error {
		t.Log("main", msg)
		// 每个handler阻塞5秒
		time.Sleep(time.Second * 5)
		return nil
	})

	go producer(ctx, "main", pubsub)

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

	pubsub := eio.NewGoPubsub(ctx, eio.GoPubsubConfig{})
	router := eio.NewRouterWithConfig(eio.RouterConfig{CloseTimeout: time.Second * 2})

	handler := router.AddHandler("1", "main", pubsub, func(msg *message.Context) error {
		t.Log("main", msg)
		// 每个handler阻塞3秒
		time.Sleep(time.Second * 3)
		return nil
	})

	go producer(ctx, "main", pubsub)

	//测试当router中的handler停止后，router是否正常关闭
	go func() {

		time.Sleep(time.Second * 4)
		handler.Stop()
		t.Log("handler stoped")
	}()

	err := router.Run(ctx)
	assert.NoError(t, err)
}
