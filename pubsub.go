package eio

import "context"

type Publisher interface {
	Publish(ctx context.Context, topic string, message Message) error
	Close()
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan Message, error)
	Close()
}
