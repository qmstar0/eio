package gopubsub

import (
	"EventDriven/message"
	"context"
	"github.com/asaskevich/EventBus"
)

type GoPubsub struct {
	name string
	bus  EventBus.Bus
}

func NewGoPubsub(name string, bus EventBus.Bus) *GoPubsub {
	return &GoPubsub{
		name: name,
		bus:  bus,
	}
}

func (g GoPubsub) Name() string {
	return g.name
}

func (g GoPubsub) Publish(topic string, messages ...*message.Message) error {
	g.bus.Publish(topic, messages)
	return nil
}

func (g GoPubsub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	var messageCh = make(chan *message.Message)

	go func() {
		<-ctx.Done()
		close(messageCh)
	}()

	err := g.bus.SubscribeAsync(topic, func(messages []*message.Message) {
		for _, msg := range messages {
			messageCh <- msg
		}
	}, false)
	if err != nil {
		return nil, err
	}
	return messageCh, nil
}

func (g GoPubsub) Close() error {
	return nil
}
