package cache

import "testing"

func Test_AVL(t *testing.T) {
	tree := &AVLTree{}

	tree.Set([]byte("1111"), 1, "aaaaa")
	tree.Set([]byte("2222"), 2, "bbbbb")

	t.Log(tree.Get([]byte("1111"), 1))
	t.Log(tree.Get([]byte("2222"), 2))
}
