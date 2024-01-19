package processor

import (
	"github.com/qmstar0/eventDriven"
	"github.com/qmstar0/eventDriven/message"
)

type Forwarder struct {
	Topic     string
	Publisher eventDriven.Publisher
}

func (f *Forwarder) middleware(fn HandlerFunc) HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		messages, err := fn(msg)
		if err != nil {
			return messages, err
		}
		if err = f.Publisher.Publish(f.Topic, messages...); err != nil {
			return messages, err
		}
		return messages, nil
	}
}

func getForwarderMiddlewares(forwarders []*Forwarder) []HandlerMiddleware {
	result := make([]HandlerMiddleware, len(forwarders))
	for i, forwarder := range forwarders {
		result[i] = forwarder.middleware
	}
	return result
}
