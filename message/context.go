package message

import (
	"context"
	"fmt"
	"time"
)

type Payload []byte
type Context struct {
	ID      string
	Payload Payload
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *Context) Ack() bool {
	return c.Nack(nil)
}

func (c *Context) Nack(err error) bool {
	select {
	case <-c.ctx.Done():
		return false
	default:
		c.ctx = context.WithValue(c.ctx, msgCtxKey, Done{err})
		c.cancel()
		return true
	}
}

func (c *Context) SetValue(k, v any) {
	c.ctx = context.WithValue(c.ctx, k, v)
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Context) Err() error {
	err, _ := c.ctx.Value(msgCtxKey).(error)
	return err
}
func (c *Context) Value(key any) any {
	return c.ctx.Value(key)
}

var msgCtxKey = &contextKey{"msgCtxKey"}

type contextKey struct{ name string }

func (k *contextKey) String() string {
	return "message.Context ctxKey:" + k.name

}

type Done struct {
	err error
}

func (e Done) Error() string {
	if e.err != nil {
		return fmt.Sprintf("message completed, but returned err: %s", e.err)
	}
	return "message completed"
}
func (e Done) Unwrap() error {
	return e.err
}
