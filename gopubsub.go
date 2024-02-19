package eio

import (
	"context"
	"errors"
	"github.com/qmstar0/eio/message"
	"sync"
)

type GoPubsubConfig struct {
	MessageChannelBuffer int
}
type GoPubsub struct {
	ctx             context.Context
	closeFn         context.CancelFunc
	config          GoPubsubConfig
	subscribers     map[string][]*subscriber
	subscribersLock sync.RWMutex
	topicLock       sync.Map
	subscribersWg   sync.WaitGroup
	closedLock      sync.Mutex
}

func NewGoPubsub(c context.Context, config GoPubsubConfig) *GoPubsub {
	ctx, cancelFunc := context.WithCancel(c)
	return &GoPubsub{
		ctx:             ctx,
		closeFn:         cancelFunc,
		config:          config,
		subscribers:     make(map[string][]*subscriber),
		subscribersLock: sync.RWMutex{},
		topicLock:       sync.Map{},
		subscribersWg:   sync.WaitGroup{},
		closedLock:      sync.Mutex{},
	}
}

func (g *GoPubsub) Publish(topic string, messageCtxs ...*message.Context) error {
	if g.isClosed() {
		return errors.New("Pub/Sub closed")
	}

	g.subscribersLock.RLock()
	defer g.subscribersLock.RUnlock()

	topicMutex, _ := g.topicLock.LoadOrStore(topic, &sync.Mutex{})
	topicMutex.(*sync.Mutex).Lock()
	defer topicMutex.(*sync.Mutex).Unlock()

	for i := range messageCtxs {
		msg := messageCtxs[i]
		g.sendMessage(topic, msg)
	}

	return nil
}

func (g *GoPubsub) sendMessage(topic string, message *message.Context) {
	subscribers := g.getSubscribersByTopic(topic)

	if len(subscribers) == 0 {
		return
	}
	for i := range subscribers {
		subscriber := subscribers[i]
		go func() {
			subscriber.sendMessageToMessageChannel(message)
		}()
	}
}

func (g *GoPubsub) Subscribe(ctx context.Context, topic string) (<-chan *message.Context, error) {
	if g.isClosed() {
		return nil, errors.New("Pub/Sub closed")
	}

	g.subscribersWg.Add(1)
	g.subscribersLock.Lock()

	subLock, _ := g.topicLock.LoadOrStore(topic, &sync.Mutex{})
	subLock.(*sync.Mutex).Lock()
	c, cancelFunc := context.WithCancel(ctx)

	s := &subscriber{
		ctx:           c,
		messageCh:     make(chan *message.Context, g.config.MessageChannelBuffer),
		messageChLock: sync.Mutex{},
	}

	go func(s *subscriber, g *GoPubsub) {
		select {
		case <-s.ctx.Done():
		case <-g.ctx.Done():
			cancelFunc()
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

	subscribersCopy := make([]*subscriber, len(subscribers))
	copy(subscribersCopy, subscribers)

	return subscribersCopy
}

func (g *GoPubsub) Close() error {
	g.closedLock.Lock()
	defer g.closedLock.Unlock()
	g.closeFn()
	g.subscribersWg.Wait()
	return nil
}

func (g *GoPubsub) isClosed() bool {
	g.closedLock.Lock()
	defer g.closedLock.Unlock()

	select {
	case <-g.ctx.Done():
		return true
	default:
		return false
	}
}

type subscriber struct {
	ctx           context.Context
	messageCh     chan *message.Context
	messageChLock sync.Mutex
}

func (s *subscriber) Close() {
	s.messageChLock.Lock()
	defer s.messageChLock.Unlock()
	close(s.messageCh)
}
func (s *subscriber) sendMessageToMessageChannel(msg *message.Context) {
	s.messageChLock.Lock()
	defer s.messageChLock.Unlock()

	select {
	case s.messageCh <- msg:
	case <-s.ctx.Done():
		return
	}
}
