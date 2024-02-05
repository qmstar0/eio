package message

import "context"

func WithContext(id string, c context.Context) *Context {
	ctx, causeFunc := context.WithCancel(c)
	return &Context{
		ID:      id,
		Header:  make(Header),
		Payload: make(Payload, 0),
		ctx:     ctx,
		cancel:  causeFunc,
	}
}

func WithPayload(id string, payload Payload) *Context {
	ctx, causeFunc := context.WithCancel(context.Background())
	return &Context{
		ID:      id,
		Header:  make(Header),
		Payload: payload,
		ctx:     ctx,
		cancel:  causeFunc,
	}
}
func WithHeader(id string, header Header) *Context {
	ctx, causeFunc := context.WithCancel(context.Background())
	return &Context{
		ID:      id,
		Header:  header,
		Payload: make(Payload, 0),
		ctx:     ctx,
		cancel:  causeFunc,
	}
}

func WithBackground(id string) *Context {
	ctx, causeFunc := context.WithCancel(context.Background())
	return &Context{
		ID:      id,
		Header:  make(Header),
		Payload: make(Payload, 0),
		ctx:     ctx,
		cancel:  causeFunc,
	}
}
