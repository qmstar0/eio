package eio

import (
	"context"
	"github.com/qmstar0/eio/message"
)

type Publisher interface {
	Publish(topic string, messages ...*message.Context) error
	Close() error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan *message.Context, error)
	Close() error
}
