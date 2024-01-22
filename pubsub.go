package eio

import (
	"context"
	"github.com/qmstar0/eio/message"
)

type Publisher interface {
	Name() string
	Publish(topic string, messages ...*message.Message) error
	Close() error
}

type Subscriber interface {
	Name() string
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
	Close() error
}
