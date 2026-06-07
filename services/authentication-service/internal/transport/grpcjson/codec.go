package grpcjson

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

const Name = "json"

type Codec struct{}

func Register() {
	encoding.RegisterCodec(Codec{})
}

func (Codec) Name() string {
	return Name
}

func (Codec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (Codec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
