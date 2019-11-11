package cache

import (
	"bytes"
)

type node struct {
	height  uint16
	hashKey uint16
	key     []byte
	val     interface{}
	left    *node
	right   *node
}

type nodeSetter struct {
	hashKey uint16
	key     []byte
	val     interface{}
	oldVal  interface{}
}

func (n *nodeSetter) set(p *node) *node {
	if p == nil {
		return &node{
			hashKey: n.hashKey,
			key:     n.key,
			val:     n.val,
		}
	}
	if n.hashKey < p.hashKey {
		p.left = n.set(p.left)
	} else if n.hashKey > p.hashKey {
		p.right = n.set(p.right)
	} else {
		if bytes.Equal(n.key, p.key) {
			n.oldVal = p.val
			p.val = n.val
		} else {
			p.left = n.set(p.left)
		}
	}
	p.height = uint16(1) + n.maxUint16(n.heightVal(p.right), n.heightVal(p.left))
	return p
}

func (n *nodeSetter) maxUint16(a uint16, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

func (n *nodeSetter) heightVal(p *node) uint16 {
	if p == nil {
		return 0
	}
	return p.height
}

type nodeGetter struct {
	hashKey uint16
	key     []byte
}

func (n *nodeGetter) get(p *node) *node {
	if p == nil {
		return nil
	}
	if n.hashKey < p.hashKey {
		return n.get(p.left)
	}
	if n.hashKey > p.hashKey {
		return n.get(p.right)
	}
	if bytes.Equal(n.key, p.key) {
		return p
	}
	return n.get(p.left)
}

type AVLTree struct {
	root *node
}

func (t *AVLTree) Set(key []byte, hashKey uint16, val interface{}) interface{} {
	n := &nodeSetter{key: key, hashKey: hashKey, val: val}
	t.root = n.set(t.root)
	return n.oldVal
}

func (t *AVLTree) Get(key []byte, hashKey uint16) interface{} {
	n := &nodeGetter{key: key, hashKey: hashKey}
	p := n.get(t.root)
	if p == nil {
		return nil
	}
	return p.val
}
