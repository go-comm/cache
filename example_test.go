package cache

import (
	"testing"
)

func Test_Memery(t *testing.T) {

	m := NewMemery()

	m.Put([]byte("user"), "admin")
	t.Log(m.Get([]byte("user")))

	m.Put([]byte("user1"), "tom")
	t.Log(m.Get([]byte("user1")))

	m.Put([]byte("user"), "guest")
	t.Log(m.Get([]byte("user")))

	t.Log(m.Get([]byte("user1")))

	t.Log(m.Get([]byte("user2")))

}
