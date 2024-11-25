package httpmock

import (
	"encoding/base64"
	"encoding/json"
)

type Body interface {
	Bytes() []byte
}

type RawBody []byte

func (ri RawBody) Bytes() []byte {
	return ri
}

type NoBody struct{}

func (NoBody) Bytes() []byte {
	return []byte{}
}

type JSON struct {
	Input any
}

func (j JSON) Bytes() []byte {
	data, err := json.Marshal(j.Input)
	if err != nil {
		panic("marshal json input, " + err.Error())
	}

	return data
}

type Base64 struct {
	Raw []byte
}

func (j Base64) Bytes() []byte {
	dst := make([]byte, len(j.Raw))

	base64.StdEncoding.Encode(dst, j.Raw)

	return dst
}
