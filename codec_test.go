package eio_test

import (
	"github.com/qmstar0/eio"
	"testing"
)

type Test struct {
	Name  string
	Age   int
	Hobby []string
	Score map[string]string
}

func TestJsonCodec(t *testing.T) {
	codec := eio.NewJSONCodec()
	msg, err := codec.Encode("hi", 1, []string{"test", "test2"})
	if err != nil {
		t.Fatal("err on encoding", err)
	}
	t.Log(msg)

	var (
		s  string
		i  int
		li []string
	)
	err = codec.Decode(msg, &s, &i, &li)
	if err != nil {
		t.Fatal("err on decoding", err)
	}
	t.Log(s, i, li)
}

func BenchmarkJsonCodec(b *testing.B) {
	codec := eio.NewJSONCodec()
	testData := Test{
		Name:  "John",
		Age:   18,
		Hobby: []string{"test1", "test2", "test3"},
		Score: map[string]string{
			"test_k1": "test_v",
			"test_k2": "test_v",
			"test_k3": "test_v",
			"test_k4": "test_v",
		},
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg, err := codec.Encode(testData, testData, testData)
			if err != nil {
				b.Fatal("err on encoding", err)
			}
			var (
				decodeData1 Test
				decodeData2 Test
				decodeData3 Test
			)
			err = codec.Decode(msg, &decodeData1, &decodeData2, &decodeData3)
			if err != nil {
				b.Fatal("err on decoding", err)
			}
		}
	})
}

func TestGobCodec(t *testing.T) {
	codec := eio.NewGobCodec()
	msg, err := codec.Encode("hi", 1, []string{"test", "test2"})
	if err != nil {
		t.Fatal("err on encoding", err)
	}
	t.Log(msg)

	var (
		s  string
		i  int
		li []string
	)
	err = codec.Decode(msg, &s, &i, &li)
	if err != nil {
		t.Fatal("err on decoding", err)
	}
	t.Log(s, i, li)
}
func BenchmarkGobCodec(b *testing.B) {
	codec := eio.NewGobCodec()
	testData := Test{
		Name:  "John",
		Age:   18,
		Hobby: []string{"test1", "test2", "test3"},
		Score: map[string]string{
			"test_k1": "test_v",
			"test_k2": "test_v",
			"test_k3": "test_v",
			"test_k4": "test_v",
		},
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg, err := codec.Encode(testData, testData, testData)
			if err != nil {
				b.Fatal("err on encoding", err)
			}
			var (
				decodeData1 Test
				decodeData2 Test
				decodeData3 Test
			)
			err = codec.Decode(msg, &decodeData1, &decodeData2, &decodeData3)
			if err != nil {
				b.Fatal("err on decoding", err)
			}
		}
	})
}
