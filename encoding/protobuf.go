package encoding

import (
	"errors"

	"github.com/golang/protobuf/proto"
)

var ProtobufEncoding = new(protobufEncoding)

type protobufEncoding struct{}

func (e protobufEncoding) Encode(buf []byte, v interface{}) ([]byte, error) {
	m, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("cache: the value must be proto.Message")
	}
	w := proto.NewBuffer(buf)
	err := w.Marshal(m)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), err
}

func (e protobufEncoding) Decode(d interface{}, b []byte) error {
	m, ok := d.(proto.Message)
	if !ok {
		return errors.New("cache: the data must be proto.Message")
	}
	var buf proto.Buffer
	buf.SetBuf(b)
	err := buf.Unmarshal(m)
	return err
}
