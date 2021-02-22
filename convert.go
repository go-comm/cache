package cache

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

func Byte(v interface{}, err error) (byte, error) {
	if err != nil {
		return 0, err
	}
	switch p := v.(type) {
	case int8:
		return byte(p), nil
	case uint8:
		return p, nil
	case int16:
		return byte(p), nil
	case uint16:
		return byte(p), nil
	case int:
		return byte(p), nil
	case int32:
		return byte(p), nil
	case uint32:
		return byte(p), nil
	case int64:
		return byte(p), nil
	case uint64:
		return byte(p), nil
	default:
		return 0, fmt.Errorf("cache: can't convert %v to byte", v)
	}
}

func Int(v interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	switch p := v.(type) {
	case int8:
		return int(p), nil
	case uint8:
		return int(p), nil
	case int16:
		return int(p), nil
	case uint16:
		return int(p), nil
	case int:
		return p, nil
	case uint:
		return int(p), nil
	case int32:
		return int(p), nil
	case uint32:
		return int(p), nil
	case int64:
		return int(p), nil
	case uint64:
		return int(p), nil
	case []byte:
		n, err := strconv.ParseInt(string(p), 10, 0)
		if err != nil {
			return 0, err
		}
		return int(n), nil
	case string:
		n, err := strconv.ParseInt(p, 10, 0)
		if err != nil {
			return 0, err
		}
		return int(n), nil
	default:
		return 0, fmt.Errorf("cache: can't convert %v to int", v)
	}
}

func Uint(v interface{}, err error) (uint, error) {
	if err != nil {
		return 0, err
	}
	switch p := v.(type) {
	case int8:
		return uint(p), nil
	case uint8:
		return uint(p), nil
	case int16:
		return uint(p), nil
	case uint16:
		return uint(p), nil
	case int:
		return uint(p), nil
	case uint:
		return p, nil
	case int32:
		return uint(p), nil
	case uint32:
		return uint(p), nil
	case int64:
		return uint(p), nil
	case uint64:
		return uint(p), nil
	case []byte:
		n, err := strconv.ParseUint(string(p), 10, 0)
		return uint(n), err
	case string:
		n, err := strconv.ParseUint(p, 10, 0)
		return uint(n), err
	default:
		return 0, fmt.Errorf("cache: can't convert %v to uint", v)
	}
}

func Int8(v interface{}, err error) (int8, error) {
	n, err := Int(v, err)
	return int8(n), err
}

func Uint8(v interface{}, err error) (uint8, error) {
	n, err := Int(v, err)
	return uint8(n), err
}

func Int16(v interface{}, err error) (int16, error) {
	n, err := Int(v, err)
	return int16(n), err
}

func Uint16(v interface{}, err error) (uint16, error) {
	n, err := Int(v, err)
	return uint16(n), err
}

func Int32(v interface{}, err error) (int32, error) {
	n, err := Int(v, err)
	return int32(n), err
}

func Uint32(v interface{}, err error) (uint32, error) {
	n, err := Int(v, err)
	return uint32(n), err
}

func Int64(v interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	switch p := v.(type) {
	case int8:
		return int64(p), nil
	case uint8:
		return int64(p), nil
	case int16:
		return int64(p), nil
	case uint16:
		return int64(p), nil
	case int:
		return int64(p), nil
	case uint:
		return int64(p), nil
	case int32:
		return int64(p), nil
	case uint32:
		return int64(p), nil
	case int64:
		return p, nil
	case uint64:
		return int64(p), nil
	case []byte:
		return strconv.ParseInt(string(p), 10, 0)
	case string:
		return strconv.ParseInt(p, 10, 0)
	default:
		return 0, fmt.Errorf("cache: can't convert %v to int64", v)
	}
}

func Uint64(v interface{}, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}
	switch p := v.(type) {
	case int8:
		return uint64(p), nil
	case uint8:
		return uint64(p), nil
	case int16:
		return uint64(p), nil
	case uint16:
		return uint64(p), nil
	case int:
		return uint64(p), nil
	case uint:
		return uint64(p), nil
	case int32:
		return uint64(p), nil
	case uint32:
		return uint64(p), nil
	case int64:
		return uint64(p), nil
	case uint64:
		return p, nil
	case []byte:
		return strconv.ParseUint(string(p), 10, 0)
	case string:
		return strconv.ParseUint(p, 10, 0)
	default:
		return 0, fmt.Errorf("cache: can't convert %v to uint64", v)
	}
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

func UnsafeConvert(dst interface{}, src interface{}) {
	refDst := reflect.ValueOf(dst)
	refSrc := reflect.ValueOf(src)

	if refDst.Kind() != reflect.Ptr {
		panic(errors.New("cache: dst must be ptr"))
	}

	refDst.Elem().Set(refSrc)
}
