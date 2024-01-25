package message

import "context"

func WithPayload(id string, payload Payload) *Context {
	ctx, causeFunc := context.WithCancelCause(context.Background())
	value := context.WithValue(ctx, setCtxKey, ctx)
	return &Context{
		ID:      id,
		Header:  make(Header),
		Payload: payload,
		ctx:     value,
		cancel:  causeFunc,
	}
}
func WithHeader(id string, header Header) *Context {
	ctx, causeFunc := context.WithCancelCause(context.Background())
	value := context.WithValue(ctx, setCtxKey, ctx)
	return &Context{
		ID:      id,
		Header:  header,
		Payload: make(Payload, 0),
		ctx:     value,
		cancel:  causeFunc,
	}
}

func WithBackground(id string) *Context {
	ctx, causeFunc := context.WithCancelCause(context.Background())
	value := context.WithValue(ctx, setCtxKey, ctx)
	return &Context{
		ID:      id,
		Header:  make(Header),
		Payload: make(Payload, 0),
		ctx:     value,
		cancel:  causeFunc,
	}
}
