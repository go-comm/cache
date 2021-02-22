package cache

import (
	"context"
	"testing"
)

func Test_Memery(t *testing.T) {
	m := NewMemery()

	m.Put(context.TODO(), []byte("user"), "admin")
	t.Log(m.Get(context.TODO(), []byte("user")))

	m.Put(context.TODO(), []byte("user1"), "tom")
	t.Log(m.Get(context.TODO(), []byte("user1")))

	m.Put(context.TODO(), []byte("user"), "guest")
	t.Log(m.Get(context.TODO(), []byte("user")))

	t.Log(m.Get(context.TODO(), []byte("user1")))

	t.Log(m.Get(context.TODO(), []byte("user2")))
}

func Test_Event(t *testing.T) {
	m := NewMemery()

	type User struct {
		ID   int
		Name string
	}

	m.Put(context.TODO(), []byte("user/1000"), JSON.WithData(&User{ID: 1000, Name: "admin"}))

	u := &User{}
	JSON.WithData2(m.Get(context.TODO(), []byte("user/1000"))).Unmarshal(u)
	t.Log(u)
}
