package cache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

var (
	JSON = WithEncoding(&stdJSONEncoding{})
	Gob  = WithEncoding(&stdGobEncoding{})
)

type Encoding interface {
	Encode([]byte, interface{}) ([]byte, error)

	Decode(interface{}, []byte) error
}

func WithEncoding(en Encoding) Event {
	return Event{en: en}
}

type stdJSONEncoding struct{}

func (e stdJSONEncoding) Encode(buf []byte, v interface{}) ([]byte, error) {
	var w = bytes.NewBuffer(buf)
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
	var w = bytes.NewBuffer(buf)
	err := gob.NewEncoder(w).Encode(v)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (e stdGobEncoding) Decode(d interface{}, b []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(b)).Decode(d)
}

type Event struct {
	err error
	ttl int64
	en  Encoding
	v   interface{}
}

func (event Event) WithData(v interface{}) *Event {
	event.v = v
	return &event
}

func (event Event) WithData2(v interface{}, err error) *Event {
	event.v = v
	event.err = err
	return &event
}

func (event Event) WithData3(v interface{}, ttl int64, err error) *Event {
	event.err = err
	event.v = v
	event.ttl = ttl
	return &event
}

func (event *Event) Marshal(buf []byte) ([]byte, error) {
	if event == nil {
		return nil, nil
	}
	if event.err != nil {
		return nil, event.err
	}
	b, err := event.en.Encode(buf, event.v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (event *Event) Unmarshal(d interface{}) error {
	if event == nil {
		return nil
	}
	if event.err != nil {
		return event.err
	}
	b, ok := event.v.([]byte)
	if !ok {
		return fmt.Errorf("cache: can't convert %v to []byte", event.v)
	}
	return event.en.Decode(d, b)
}
