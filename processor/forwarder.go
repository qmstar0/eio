package processor

import (
	"github.com/qmstar0/eventDriven"
	"github.com/qmstar0/eventDriven/message"
)

type Forwarder struct {
	topic     string
	publisher eventDriven.Publisher
}

func Forward(topic string, pub eventDriven.Publisher) Forwarder {
	return Forwarder{
		topic:     topic,
		publisher: pub,
	}
}

func (f *Forwarder) Middleware(fn HandlerFunc) HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		messages, err := fn(msg)
		if err != nil {
			return messages, err
		}
		if err = f.publisher.Publish(f.topic, messages...); err != nil {
			return messages, err
		}
		return messages, nil
	}
}

func getForwarderMiddlewares(forwarders []Forwarder) []HandlerMiddleware {
	result := make([]HandlerMiddleware, len(forwarders))
	for i, forwarder := range forwarders {
		result[i] = forwarder.Middleware
	}
	return result
}
