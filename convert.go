package cache

import (
	"bytes"
	"fmt"
)

func Byte(v interface{}, err error) (byte, error) {
	if err != nil {
		return 0, err
	}
	o, ok := v.(byte)
	if !ok {
		return 0, fmt.Errorf("cache: can't convert %v to byte", v)
	}
	return o, nil
}

func Int8(v interface{}, err error) (int8, error) {
	if err != nil {
		return 0, err
	}
	o, ok := v.(int8)
	if !ok {
		return 0, fmt.Errorf("cache: can't convert %v to int8", v)
	}
	return o, nil
}

func Int16(v interface{}, err error) (int16, error) {
	if err != nil {
		return 0, err
	}
	o, ok := v.(int16)
	if !ok {
		return 0, fmt.Errorf("cache: can't convert %v to int16", v)
	}
	return o, nil
}

func Int(v interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	o, ok := v.(int)
	if !ok {
		return 0, fmt.Errorf("cache: can't convert %v to int", v)
	}
	return o, nil
}

func Int32(v interface{}, err error) (int32, error) {
	if err != nil {
		return 0, err
	}
	o, ok := v.(int32)
	if !ok {
		return 0, fmt.Errorf("cache: can't convert %v to int32", v)
	}
	return o, nil
}

func Int64(v interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	o, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("cache: can't convert %v to int64", v)
	}
	return o, nil
}

func Ints(v interface{}, err error) ([]int, error) {
	if err != nil {
		return nil, err
	}
	o, ok := v.([]int)
	if !ok {
		return nil, fmt.Errorf("cache: can't convert %v to []int", v)
	}
	return o, nil
}

func Int64s(v interface{}, err error) ([]int64, error) {
	if err != nil {
		return nil, err
	}
	o, ok := v.([]int64)
	if !ok {
		return nil, fmt.Errorf("cache: can't convert %v to []int64", v)
	}
	return o, nil
}

func Bytes(v interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	o, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("cache: can't convert %v to []byte", v)
	}
	return o, nil
}

func String(v interface{}, err error) (string, error) {
	if err != nil {
		return "", err
	}
	o, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("cache: can't convert %v to string", v)
	}
	return o, nil
}

func Strings(v interface{}, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	o, ok := v.([]string)
	if !ok {
		return nil, fmt.Errorf("cache: can't convert %v to []string", v)
	}
	return o, nil
}

func Buffer(v interface{}, err error) (*bytes.Buffer, error) {
	if err != nil {
		return nil, err
	}
	o, ok := v.(*bytes.Buffer)
	if !ok {
		return nil, fmt.Errorf("cache: can't convert %v to *bytes.Buffer", v)
	}
	return o, nil
}
