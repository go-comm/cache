package cache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

type Encoding interface {
	Encode([]byte, interface{}) ([]byte, error)

	Decode(interface{}, []byte) error
}

type stdJSONEncoding struct{}

func (e stdJSONEncoding) Encode(buf []byte, v interface{}) ([]byte, error) {
	w := bytes.NewBuffer(buf)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (e stdJSONEncoding) Decode(d interface{}, b []byte) error {
	return json.Unmarshal(b, d)
}

type stdGobEncoding struct{}

func (e stdGobEncoding) Encode(buf []byte, v interface{}) ([]byte, error) {
	w := bytes.NewBuffer(buf)
	err := gob.NewEncoder(w).Encode(v)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (e stdGobEncoding) Decode(d interface{}, b []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(b)).Decode(d)
}
