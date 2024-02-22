package eio

type Codec interface {
	Encoder
	Decoder
}

type Encoder interface {
	Encode(inputs ...any) (Message, error)
}

type Decoder interface {
	Decode(msg Message, outputs ...any) error
}
