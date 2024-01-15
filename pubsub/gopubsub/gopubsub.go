// 原文件地址:https://github.com/ThreeDotsLabs/watermill/blob/master/pubsub/gochannel/pubsub.go

package gopubsub

import (
	"context"
	"errors"
	"github.com/qmstar0/eventRouter/message"
	"sync"
)

type GoPubsubConfig struct {
	MessageChannelBuffer int64
}
type GoPubsub struct {
	name   string
	config GoPubsubConfig

	subscribers     map[string][]*subscriber
	subscribersLock sync.RWMutex
	topicLock       sync.Map
	subscribersWg   sync.WaitGroup

	closing    chan struct{}
	closed     bool
	closedLock sync.Mutex
}

func NewGoPubsub(name string, config GoPubsubConfig) *GoPubsub {
	return &GoPubsub{
		name:            name,
		config:          config,
		subscribers:     make(map[string][]*subscriber),
		subscribersLock: sync.RWMutex{},
		topicLock:       sync.Map{},
		subscribersWg:   sync.WaitGroup{},
		closing:         make(chan struct{}),
		closed:          false,
		closedLock:      sync.Mutex{},
	}
}

func (g *GoPubsub) Name() string {
	return g.name
}

func (g *GoPubsub) Publish(topic string, messages ...*message.Message) error {
	if g.isClosed() {
		return errors.New("Pub/Sub closed")
	}

	messagesCopy := make([]*message.Message, len(messages))
	for i, msg := range messages {
		messagesCopy[i] = msg.Copy()
	}

	g.subscribersLock.RLock()
	defer g.subscribersLock.RUnlock()

	topicMutex, _ := g.topicLock.LoadOrStore(topic, &sync.Mutex{})
	topicMutex.(*sync.Mutex).Lock()
	defer topicMutex.(*sync.Mutex).Unlock()

	for i := range messagesCopy {
		msg := messagesCopy[i]

		g.sendMessage(topic, msg)

	}

	return nil
}

func (g *GoPubsub) sendMessage(topic string, message *message.Message) {
	subscribers := g.getSubscribersByTopic(topic)

	if len(subscribers) == 0 {
		return
	}
	//wg := &sync.WaitGroup{}
	for i := range subscribers {
		subscriber := subscribers[i]
		//wg.Add(1)
		go func() {
			subscriber.sendMessageToMessageChannel(message)
			//wg.Done()
		}()
	}
	//wg.Wait()
}

func (g *GoPubsub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	g.closedLock.Lock()

	if g.closed {
		g.closedLock.Unlock()
		return nil, errors.New("Pub/Sub closed")
	}

	g.subscribersWg.Add(1)
	g.closedLock.Unlock()

	g.subscribersLock.Lock()

	subLock, _ := g.topicLock.LoadOrStore(topic, &sync.Mutex{})
	subLock.(*sync.Mutex).Lock()

	s := &subscriber{
		ctx:           ctx,
		messageCh:     make(chan *message.Message, g.config.MessageChannelBuffer),
		messageChLock: sync.Mutex{},

		closed:  false,
		closing: make(chan struct{}),
	}

	go func(s *subscriber, g *GoPubsub) {
		select {
		case <-ctx.Done():
			// unblock
		case <-g.closing:
			// unblock
		}

		s.Close()

		g.subscribersLock.Lock()
		defer g.subscribersLock.Unlock()

		subLock, _ := g.topicLock.Load(topic)
		subLock.(*sync.Mutex).Lock()
		defer subLock.(*sync.Mutex).Unlock()

		g.removeSubscriber(topic, s)
		g.subscribersWg.Done()
	}(s, g)

	go func(s *subscriber) {
		defer g.subscribersLock.Unlock()
		defer subLock.(*sync.Mutex).Unlock()

		g.addSubscriber(topic, s)
	}(s)

	return s.messageCh, nil
}

func (g *GoPubsub) addSubscriber(topic string, s *subscriber) {
	if _, ok := g.subscribers[topic]; !ok {
		g.subscribers[topic] = make([]*subscriber, 0)
	}
	g.subscribers[topic] = append(g.subscribers[topic], s)
}

func (g *GoPubsub) removeSubscriber(topic string, toRemove *subscriber) {
	for i, sub := range g.subscribers[topic] {
		if sub == toRemove {
			g.subscribers[topic] = append(g.subscribers[topic][:i], g.subscribers[topic][i+1:]...)
			break
		}
	}
}

func (g *GoPubsub) getSubscribersByTopic(topic string) []*subscriber {
	subscribers, ok := g.subscribers[topic]
	if !ok {
		return nil
	}

	// let's do a copy to avoid race conditions and deadlocks due to lock
	subscribersCopy := make([]*subscriber, len(subscribers))
	copy(subscribersCopy, subscribers)

	return subscribersCopy
}

func (g *GoPubsub) Close() error {

	g.closedLock.Lock()
	defer g.closedLock.Unlock()

	g.closed = true
	close(g.closing)

	g.subscribersWg.Wait()
	return nil
}

func (g *GoPubsub) isClosed() bool {
	g.closedLock.Lock()
	defer g.closedLock.Unlock()

	return g.closed
}
