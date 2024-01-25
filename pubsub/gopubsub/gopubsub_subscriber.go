package gopubsub

import (
	"context"
	"github.com/qmstar0/eio/message"
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

	select {
	case s.messageCh <- msg.Copy():
	case <-s.closing:
		return
	}
}
