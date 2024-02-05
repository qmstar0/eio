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

func TestContext_SetValue(t *testing.T) {

	msg := message.WithPayload(eio.NewUUID(), []byte("hi"))

	msg.SetValue(1, "test")

	assert.Equal(t, msg.Value(1), "test")
}

func TestContext_DoneAndErr(t *testing.T) {
	t.Run("ack", func(t *testing.T) {
		msg := message.WithPayload(eio.NewUUID(), []byte("hi"))
		go func() {
			select {
			case <-msg.Done():
				assert.NotEqual(t, msg.Err(), context.Canceled)
				assert.Equal(t, errors.Unwrap(msg.Err()), nil)
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
				assert.NotEqual(t, msg.Err(), err)
				assert.Equal(t, errors.Unwrap(msg.Err()), err)
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
	msg := message.WithContext(eio.NewUUID(), ctx)
	<-msg.Done()
	assert.Equal(t, context.Cause(msg), context.DeadlineExceeded)
}

func TestContext_Cause(t *testing.T) {
	err := errors.New("test err")
	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second, err)
	defer cancel()

	msg := message.WithContext(eio.NewUUID(), ctx)
	<-msg.Done()
	assert.Equal(t, context.Cause(msg), err)
}
