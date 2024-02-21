package eio

import (
	"bytes"
	"encoding/json"
	"sync"
)

type jsonCodec struct {
	rbuf *bytes.Buffer
	wbuf *bytes.Buffer

	enc *json.Encoder
	dec *json.Decoder

	rmx, wmx sync.Mutex
}

func NewJSONCodec() Codec {
	rbuf, wbuf := bytes.NewBuffer(make([]byte, 0, 4096)), bytes.NewBuffer(make([]byte, 0, 4096))
	return &jsonCodec{
		rbuf: rbuf,
		wbuf: wbuf,
		dec:  json.NewDecoder(rbuf),
		enc:  json.NewEncoder(wbuf),
		rmx:  sync.Mutex{},
		wmx:  sync.Mutex{},
	}
}

func (j *jsonCodec) Encode(inputs ...any) (Message, error) {
	j.wmx.Lock()
	defer j.wmx.Unlock()
	defer j.wbuf.Reset()

	for i := range inputs {
		err := j.enc.Encode(inputs[i])
		if err != nil {
			return nil, err
		}
	}
	return j.wbuf.Bytes(), nil
}

func (j *jsonCodec) Decode(msg Message, outputs ...any) error {
	j.rmx.Lock()
	defer j.rmx.Unlock()
	defer j.rbuf.Reset()

	j.rbuf.Write(msg)
	for i := range outputs {
		err := j.dec.Decode(outputs[i])
		if err != nil {
			return err
		}
	}
	return nil
}
