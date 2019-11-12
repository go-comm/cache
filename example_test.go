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

	t.Log(m.Get([]byte("user2")))

	var es []*Entry
	es = make([]*Entry, 0, 15)
	es = m.List(es)
	for _, e := range es {
		t.Log(e.k, e.LoadValue())
	}

	m.Del([]byte("user"))

	es = es[:0]
	es = m.List(es)
	for _, e := range es {
		t.Log(e.k, e.LoadValue())
	}

}
