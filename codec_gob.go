package eio

import (
	"bytes"
	"encoding/gob"
	"sync"
)

type gobCodec struct {
	rbuf *bytes.Buffer
	wbuf *bytes.Buffer

	enc *gob.Encoder
	dec *gob.Decoder

	rmx, wmx sync.Mutex
}

func NewGobCodec() Codec {
	rbuf, wbuf := bytes.NewBuffer(make([]byte, 0, 4096)), bytes.NewBuffer(make([]byte, 0, 4096))

	return &gobCodec{
		rbuf: rbuf,
		wbuf: wbuf,
		dec:  gob.NewDecoder(rbuf),
		enc:  gob.NewEncoder(wbuf),
		rmx:  sync.Mutex{},
		wmx:  sync.Mutex{},
	}
}

func (j *gobCodec) Encode(inputs ...any) (Message, error) {
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

func (j *gobCodec) Decode(msg Message, outputs ...any) error {
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
