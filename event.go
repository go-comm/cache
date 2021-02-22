package cache

import "fmt"

var (
	JSON = WithEncoding(&stdJSONEncoding{})
	Gob  = WithEncoding(&stdGobEncoding{})
)

func WithEncoding(en Encoding) *Event {
	return &Event{en: en}
}

type Event struct {
	err error
	ttl int64
	en  Encoding
	v   interface{}
}

func (event *Event) Dump() *Event {
	return &Event{event.err, event.ttl, event.en, event.v}
}

func (event *Event) WithData(v interface{}) *Event {
	e := event.Dump()
	e.v = v
	return e
}

func (event *Event) WithData2(v interface{}, err error) *Event {
	e := event.Dump()
	e.v = v
	e.err = err
	return e
}

func (event *Event) WithData3(v interface{}, ttl int64, err error) *Event {
	e := event.Dump()
	e.v = v
	e.ttl = ttl
	e.err = err
	return e
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
