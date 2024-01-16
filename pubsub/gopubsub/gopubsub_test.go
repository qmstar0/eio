package gopubsub_test

import (
	"context"
	"github.com/qmstar0/eventRouter/message"
	"github.com/qmstar0/eventRouter/pubsub"
	"github.com/qmstar0/eventRouter/pubsub/gopubsub"
	"testing"
	"time"
)

var TestRunDuration = time.Second * 5

func Publisher(t *testing.T, pub pubsub.Publisher, ctx context.Context) {
	var count = 0
	for {
		select {
		case <-ctx.Done():
			t.Logf("context.Done(); 共计发布%d次", count)
			return
		default:
			if err := pub.Publish(
				"pub_test",
				message.NewMessage(gopubsub.NewUUID(), []byte("hi"))); err != nil {
				t.Logf("发布时发生错误:%s", err)
				t.Logf("context.Done(); 共计发布%d次", count)
				return
			}
			count++
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func TestGoPubsub(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestRunDuration)
	defer cancel()
	goPubsub := gopubsub.NewGoPubsub("test", gopubsub.GoPubsubConfig{})

	go Publisher(t, goPubsub, ctx)

	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	messageCh, err := goPubsub.Subscribe(subCtx, "pub_test")
	if err != nil {
		t.Fatal(err)
	}

	for msg := range messageCh {
		t.Log("收到消息:", msg)
	}
	time.Sleep(time.Millisecond * 500)
}

func TestGoPubsub_Close(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestRunDuration)
	defer cancel()
	goPubsub := gopubsub.NewGoPubsub("test", gopubsub.GoPubsubConfig{})

	go Publisher(t, goPubsub, ctx)

	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	messageCh, err := goPubsub.Subscribe(subCtx, "pub_test")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(time.Second * 3)
		err = goPubsub.Close()
		if err != nil {
			t.Error(err)
			return
		} else {
			t.Log("正常关闭")
		}
	}()

	for msg := range messageCh {
		t.Log("收到消息:", msg)
	}

	time.Sleep(time.Millisecond * 500)
}
