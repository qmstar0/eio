package message

import (
	"context"
	"time"
)

type Payload []byte
type Context struct {
	ID string

	Header Header

	Payload Payload

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (c *Context) Ack() bool {
	select {
	case <-c.ctx.Done():
		return false
	default:
		c.cancel(nil)
		return true
	}
}

func (c *Context) Nack(err error) bool {
	select {
	case <-c.ctx.Done():
		return false
	default:
		c.cancel(err)
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
	return context.Cause(c.ctx)
}

func (c *Context) Value(key any) any {
	return c.ctx.Value(key)
}
