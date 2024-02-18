package message

import (
	"context"
)

func WithContext(id string, c context.Context) *Context {
	ctx, cc := context.WithCancel(c)
	return &Context{
		ID:      id,
		Payload: make(Payload, 0),
		ctx:     ctx,
		cancel:  cc,
	}
}
func WithPayload(id string, payload Payload) *Context {
	ctx, cc := context.WithCancel(context.Background())
	return &Context{
		ID:      id,
		Payload: payload,
		ctx:     ctx,
		cancel:  cc,
	}
}
