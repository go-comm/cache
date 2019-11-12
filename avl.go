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

type nodeIterator struct {
	fn func(interface{}) bool
}

func (n *nodeIterator) iterate(p *node) bool {
	if p == nil {
		return true
	}
	if p.left != nil {
		if !n.iterate(p.left) {
			return false
		}
	}
	if p.right != nil {
		if !n.iterate(p.right) {
			return false
		}
	}
	return n.fn(p.val)
}

type nodeDeleter struct {
	hashKey uint16
	key     []byte
}

func (n *nodeDeleter) findMax(p *node) *node {
	if p == nil {
		return nil
	}
	for p.right != nil {
		p = p.right
	}
	return p
}

func (n *nodeDeleter) del(p *node) *node {
	if p == nil {
		return nil
	}
	if n.hashKey < p.hashKey {
		p.left = n.del(p.left)
	} else if n.hashKey > p.hashKey {
		p.right = n.del(p.right)
	} else if !bytes.Equal(n.key, p.key) {
		p.left = n.del(p.left)
	} else {
		if p.left != nil && p.right != nil {
			tmp := n.findMax(p.left)
			p.key = tmp.key
			p.hashKey = tmp.hashKey
			p.val = tmp.val
			n.key = tmp.key
			n.hashKey = tmp.hashKey
			p.left = n.del(p.left)
		} else if p.left == nil {
			p = p.right
		} else if p.right == nil {
			p = p.left
		} else {
			p = nil
		}
	}
	return p
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

func (t *AVLTree) Iterator(fn func(interface{}) bool) {
	n := &nodeIterator{fn: fn}
	n.iterate(t.root)
}

func (t *AVLTree) Del(key []byte, hashKey uint16) {
	n := &nodeDeleter{key: key, hashKey: hashKey}
	t.root = n.del(t.root)
}
