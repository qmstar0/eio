package gopubsub

import (
	"context"
	"github.com/qmstar0/eio/message"
	"sync"
)

type subscriber struct {
	ctx context.Context

	messageCh     chan *message.Context
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
func (s *subscriber) sendMessageToMessageChannel(msg *message.Context) {
	s.messageChLock.Lock()
	defer s.messageChLock.Unlock()

	select {
	case s.messageCh <- msg:
	case <-s.closing:
		return
	}
}
