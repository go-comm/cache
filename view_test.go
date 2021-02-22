package cache

import (
	"context"
	"testing"
)

func Test_UnsafeViewEx(t *testing.T) {
	m := NewMemery()

	type User struct {
		Name string
		Age  int
	}

	key := []byte("user/10000")
	var user *User

	UnsafeViewEx(context.TODO(), key, &user, -1, m, func() (interface{}, error) {
		return &User{
			Name: "10000",
			Age:  99,
		}, nil
	})

	t.Log(user)
}
