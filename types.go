package cache

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
)

// AnyValuer returns a Valuer that passes the value through as-is.
//
// Usage: cache.AnyValuer(&user)
func AnyValuer(ptr interface{}) Valuer {
	return &anyValuer{Ptr: ptr}
}

type anyValuer struct {
	Ptr interface{}
}

func (v *anyValuer) Value() (driver.Value, error) {
	return v.Ptr, nil
}

// AnyScanner returns a Scanner that assigns the cached value to ptr using reflection.
//
// Usage: cache.AnyScanner(&user)
func AnyScanner(ptr interface{}) Scanner {
	return &anyScanner{Ptr: ptr}
}

type anyScanner struct {
	Ptr interface{}
}

func (s *anyScanner) Scan(v interface{}) error {
	return UnsafeAssign(s.Ptr, v)
}

// EncodeValuer returns a Valuer that encodes the value as JSON (default) into bytes.
//
// Usage: cache.EncodeValuer(&user)
func EncodeValuer(ptr interface{}) Valuer {
	return &encodeValuer{Ptr: ptr}
}

type encodeValuer struct {
	Marshal func(v interface{}) ([]byte, error)
	Ptr     interface{}
}

func (v *encodeValuer) Value() (driver.Value, error) {
	m := v.Marshal
	if m == nil {
		m = json.Marshal
	}
	return m(v.Ptr)
}

// DecodeScanner returns a Scanner that decodes the cached value (JSON by default) into ptr.
//
// Usage: cache.DecodeScanner(&user)
func DecodeScanner(ptr interface{}) Scanner {
	return &decodeScanner{Ptr: ptr}
}

type decodeScanner struct {
	Unmarshal func(data []byte, v interface{}) error
	Ptr       interface{}
}

func (s *decodeScanner) Scan(v interface{}) error {
	um := s.Unmarshal
	if um == nil {
		um = json.Unmarshal
	}
	var b []byte
	switch d := v.(type) {
	case string:
		b = []byte(d)
	case []byte:
		b = d
	default:
		return fmt.Errorf("cache: unsupported type %T for decodeScanner", v)
	}
	return um(b, s.Ptr)
}

// IntScanner returns a Scanner that scans the cached value into a *int.
//
// Usage: cache.IntScanner(&count)
func IntScanner(ptr *int) Scanner {
	return &intScanner{Ptr: ptr}
}

type intScanner struct {
	Ptr *int
}

func (s *intScanner) Scan(v interface{}) error {
	switch d := v.(type) {
	case int:
		*s.Ptr = d
	case int64:
		*s.Ptr = int(d)
	case uint64:
		*s.Ptr = int(d)
	case float64:
		*s.Ptr = int(d)
	case string:
		n, err := strconv.Atoi(d)
		if err != nil {
			return err
		}
		*s.Ptr = n
	case []byte:
		n, err := strconv.Atoi(string(d))
		if err != nil {
			return err
		}
		*s.Ptr = n
	default:
		return fmt.Errorf("cache: unsupported type %T for intScanner", v)
	}
	return nil
}

// Int64Scanner returns a Scanner that scans the cached value into a *int64.
//
// Usage: cache.Int64Scanner(&count)
func Int64Scanner(ptr *int64) Scanner {
	return &int64Scanner{Ptr: ptr}
}

type int64Scanner struct {
	Ptr *int64
}

func (s *int64Scanner) Scan(v interface{}) error {
	switch d := v.(type) {
	case int64:
		*s.Ptr = d
	case int:
		*s.Ptr = int64(d)
	case uint64:
		*s.Ptr = int64(d)
	case float64:
		*s.Ptr = int64(d)
	case string:
		n, err := strconv.ParseInt(d, 10, 64)
		if err != nil {
			return err
		}
		*s.Ptr = n
	case []byte:
		n, err := strconv.ParseInt(string(d), 10, 64)
		if err != nil {
			return err
		}
		*s.Ptr = n
	default:
		return fmt.Errorf("cache: unsupported type %T for int64Scanner", v)
	}
	return nil
}

// Uint64Scanner returns a Scanner that scans the cached value into a *uint64.
//
// Usage: cache.Uint64Scanner(&count)
func Uint64Scanner(ptr *uint64) Scanner {
	return &uint64Scanner{Ptr: ptr}
}

type uint64Scanner struct {
	Ptr *uint64
}

func (s *uint64Scanner) Scan(v interface{}) error {
	switch d := v.(type) {
	case uint64:
		*s.Ptr = d
	case int:
		*s.Ptr = uint64(d)
	case int64:
		*s.Ptr = uint64(d)
	case float64:
		*s.Ptr = uint64(d)
	case string:
		n, err := strconv.ParseUint(d, 10, 64)
		if err != nil {
			return err
		}
		*s.Ptr = n
	case []byte:
		n, err := strconv.ParseUint(string(d), 10, 64)
		if err != nil {
			return err
		}
		*s.Ptr = n
	default:
		return fmt.Errorf("cache: unsupported type %T for uint64Scanner", v)
	}
	return nil
}

// BoolScanner returns a Scanner that scans the cached value into a *bool.
//
// Usage: cache.BoolScanner(&enabled)
func BoolScanner(ptr *bool) Scanner {
	return &boolScanner{Ptr: ptr}
}

type boolScanner struct {
	Ptr *bool
}

func (s *boolScanner) Scan(v interface{}) error {
	switch d := v.(type) {
	case bool:
		*s.Ptr = d
	case int:
		*s.Ptr = d != 0
	case int64:
		*s.Ptr = d != 0
	case uint64:
		*s.Ptr = d != 0
	case float64:
		*s.Ptr = d != 0
	case string:
		b, err := strconv.ParseBool(d)
		if err != nil {
			return err
		}
		*s.Ptr = b
	case []byte:
		b, err := strconv.ParseBool(string(d))
		if err != nil {
			return err
		}
		*s.Ptr = b
	default:
		return fmt.Errorf("cache: unsupported type %T for boolScanner", v)
	}
	return nil
}

// StringScanner returns a Scanner that scans the cached value into a *string.
//
// Usage: cache.StringScanner(&name)
func StringScanner(ptr *string) Scanner {
	return &stringScanner{Ptr: ptr}
}

type stringScanner struct {
	Ptr *string
}

func (s *stringScanner) Scan(v interface{}) error {
	switch d := v.(type) {
	case string:
		*s.Ptr = d
	case []byte:
		*s.Ptr = string(d)
	default:
		return fmt.Errorf("cache: unsupported type %T for stringScanner", v)
	}
	return nil
}
