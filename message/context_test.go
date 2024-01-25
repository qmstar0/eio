package message_test

import (
	"context"
	"errors"
	"github.com/qmstar0/eio"
	"github.com/qmstar0/eio/message"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestContext_SetContext(t *testing.T) {

	todo := context.TODO()

	msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
	msg.SetContext(todo)

	ctx := msg.Context()

	assert.Equal(t, &ctx, &todo)

}

func TestContext_Custom(t *testing.T) {
	msgCtx := message.WithBackground(eio.NewUUID())

	msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
	msg.SetContext(msgCtx)

	ctx, ok := msg.Context().(*message.Context)
	assert.True(t, ok)

	assert.Equal(t, ctx.ID, msgCtx.ID)
}

func TestContext_SetValue(t *testing.T) {

	msgCtx := message.WithBackground(eio.NewUUID())

	msg := message.WithPayload(eio.NewUUID(), []byte("hi"))

	msg.SetContext(msgCtx)

	msg.SetValue(1, "test")

	m, ok := msg.Context().(*message.Context)
	assert.True(t, ok)
	assert.Equal(t, m.ID, msgCtx.ID)
	assert.Equal(t, msg.Value(1), "test")
}

func TestContext_DoneAndErr(t *testing.T) {
	t.Run("ack", func(t *testing.T) {
		msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
		go func() {
			select {
			case <-msg.Done():
				assert.Equal(t, msg.Err(), context.Canceled)
			}
		}()

		msg.Ack()
	})

	t.Run("nack", func(t *testing.T) {
		err := errors.New("test err")
		msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
		go func() {
			select {
			case <-msg.Done():
				assert.Equal(t, msg.Err(), err)
			}
		}()

		msg.Nack(err)
	})
}

func TestContext_Header(t *testing.T) {
	msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
	msg.Header.Set("testkey", "testvalue")
	assert.Equal(t, msg.Header.Get("testkey"), "testvalue")
}

func TestContext_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
	msg.SetContext(ctx)
	<-msg.Done()
	assert.Equal(t, context.Cause(msg), context.DeadlineExceeded)
}

func TestContext_Cause(t *testing.T) {
	err := errors.New("test err")
	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second, err)
	defer cancel()

	msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
	msg.SetContext(ctx)
	<-msg.Done()
	assert.Equal(t, context.Cause(msg), err)
}
