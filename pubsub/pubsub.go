package pubsub

import (
	"EventDriven/message"
	"context"
)

type Publisher interface {
	Name() string
	Publish(topic string, messages ...*message.Message) error
	Close() error
}

type Subscriber interface {
	Name() string
	// Subscribe ctx应当可以停止Subscribe返回的<-chan *Message的运行，否则会发生错误，我曾犯过这个错误
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
	Close() error
}
