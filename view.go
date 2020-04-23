package cache

import (
	"errors"
)

func View(k []byte, c Cache, fn func() (interface{}, error)) (interface{}, error) {
	v, err := c.Get(k)
	if err == nil {
		return v, nil
	}
	if fn == nil {
		return nil, errors.New("function is nil")
	}
	v, err = fn()
	if err != nil {
		return nil, err
	}
	c.Put(k, v)
	return v, nil
}

func ViewEx(k []byte, ex int64, c Cache, fn func() (interface{}, error)) (interface{}, error) {
	v, err := c.Get(k)
	if err == nil {
		return v, nil
	}
	if fn == nil {
		return nil, errors.New("function is nil")
	}
	v, err = fn()
	if err != nil {
		return nil, err
	}
	c.PutEx(k, v, ex)
	return v, nil
}
