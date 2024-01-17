package gopubsub

import (
	"context"
	"github.com/qmstar0/eventDriven/message"
	"sync"
)

type subscriber struct {
	ctx context.Context

	messageCh     chan *message.Message
	messageChLock sync.Mutex

	closing chan struct{}
	closed  bool
}

func (s *subscriber) Close() {
	if s.closed {
		return
	}
	close(s.closing)

	s.messageChLock.Lock()
	defer s.messageChLock.Unlock()

	s.closed = true

	close(s.messageCh)
}
func (s *subscriber) sendMessageToMessageChannel(msg *message.Message) {
	s.messageChLock.Lock()
	defer s.messageChLock.Unlock()

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	msgCopy := msg.Copy()
	msgCopy.SetContext(ctx)

	select {
	case s.messageCh <- msgCopy:
	case <-s.closing:
		return
	}
}
