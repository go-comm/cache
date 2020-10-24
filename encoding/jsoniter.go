package encoding

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"
)

var JsoniterEncoding = new(jsoniterEncoding)

type jsoniterEncoding struct{}

func (e jsoniterEncoding) Encode(buf []byte, v interface{}) ([]byte, error) {
	var w = bytes.NewBuffer(buf)
	err := jsoniter.ConfigFastest.NewEncoder(w).Encode(v)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (e jsoniterEncoding) Decode(d interface{}, b []byte) error {
	return jsoniter.ConfigFastest.Unmarshal(b, d)
}
