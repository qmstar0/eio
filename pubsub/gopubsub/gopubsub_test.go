package gopubsub_test

import (
	"context"
	"github.com/qmstar0/eventDriven"
	"github.com/qmstar0/eventDriven/message"
	"github.com/qmstar0/eventDriven/pubsub/gopubsub"
	"sync"
	"testing"
	"time"
)

var TestRunDuration = time.Second * 3

func Publisher(t *testing.T, pub eventDriven.Publisher, ctx context.Context) {
	var count = 0
	for {
		select {
		case <-ctx.Done():
			t.Logf("context.Done(); 共计发布%d次", count)
			return
		default:
			if err := pub.Publish(
				"pub_test",
				message.NewMessage(eventDriven.NewUUID(), []byte("hi"))); err != nil {
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

func TestPublishers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestRunDuration)
	defer cancel()
	goPubsub := gopubsub.NewGoPubsub("test", gopubsub.GoPubsubConfig{})

	go Publisher(t, goPubsub, ctx)

	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	messageCh1, err := goPubsub.Subscribe(subCtx, "pub_test")
	messageCh2, err := goPubsub.Subscribe(subCtx, "pub_test")
	if err != nil {
		t.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	go func() {
		wg.Add(1)
		for msg := range messageCh1 {
			t.Log("1收到消息:", msg)
		}
		wg.Done()
	}()

	go func() {
		wg.Add(1)
		for msg := range messageCh2 {
			t.Log("2收到消息:", msg)
		}
		wg.Done()
	}()

	time.Sleep(time.Millisecond * 500)
	wg.Wait()
}
