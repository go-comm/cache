package timingwheel

import (
	"bytes"
	"container/list"
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sync/errgroup"
)

const (
	defaultMinTickDuration  = time.Millisecond * 20
	defaultMinTicksPreWheel = 1 << 8
	defaultMaxTicksPreWheel = 1<<16 - 1
	_ContextKeyToken        = "_TW_TOKEN_$"
	_TokenSize              = 16
)

type Future interface {
	Cancel()
	Token() []byte
}

var emptyFuture = &future{}

type future struct {
	p     unsafe.Pointer
	token []byte
}

func (f *future) Cancel() {
	p := f.p
	if p != nil {
		node := *(**timeoutNode)(p)
		atomic.StoreInt32(&node.canceled, 1)
	}
}
func (f *future) Token() []byte {
	return f.token
}

type TimingWheel interface {
	PostDelayed(ctx context.Context, callback func(context.Context) error, delay time.Duration) Future
	PostAtTime(ctx context.Context, callback func(context.Context) error, nanotime int64) Future
	Remove(token []byte)
	Stop()
}

func New(tickDuration time.Duration, ticksPerWheel int) TimingWheel {
	ctx := context.Background()
	tw := &timingWheel{}
	tw.group, ctx = errgroup.WithContext(ctx)
	tw.startTime = time.Now().UnixNano()
	tw.tickDuration = tickDuration
	if ticksPerWheel < defaultMinTicksPreWheel {
		ticksPerWheel = defaultMinTicksPreWheel
	}
	if ticksPerWheel > defaultMaxTicksPreWheel {
		ticksPerWheel = defaultMaxTicksPreWheel
	}
	tw.ticksPerWheel = RoundUp(ticksPerWheel)
	tw.stopChan = make(chan struct{}, 1)
	tw.mask = int64(tw.ticksPerWheel - 1)
	tw.buckets = make([]*bucket, ticksPerWheel)

	for i := 0; i < len(tw.buckets); i++ {
		tw.buckets[i] = &bucket{nodes: list.New(), wheel: tw}
	}

	go tw.runInLoop()
	return tw
}

type bucket struct {
	_     [7]uint64
	mutex sync.RWMutex
	nodes *list.List
	wheel *timingWheel
}

func (b *bucket) add(node *timeoutNode) {
	b.nodes.PushBack(node)
}

func (b *bucket) calls(timeout int64) {
	var peddingTimeoutNodes []*list.Element
	b.mutex.RLock()
	for e := b.nodes.Front(); e != nil; e = e.Next() {
		node := e.Value.(*timeoutNode)
		if node.isCanceled() || node.timeout < timeout {
			peddingTimeoutNodes = append(peddingTimeoutNodes, e)
		}
	}
	b.mutex.RUnlock()

	if len(peddingTimeoutNodes) > 0 {
		b.mutex.Lock()
		for _, e := range peddingTimeoutNodes {
			b.nodes.Remove(e)
		}
		b.mutex.Unlock()

		for _, e := range peddingTimeoutNodes {
			node := e.Value.(*timeoutNode)
			if !node.isCanceled() {
				b.call(node)
			}
		}
	}
}

func (b *bucket) call(node *timeoutNode) {
	callback := node.callback
	ctx := node.ctx
	if callback == nil {
		return
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
		default:
			b.wheel.group.Go(func() error { return callback(ctx) })
		}
	}
}

type timeoutNode struct {
	ctx      context.Context
	token    []byte
	canceled int32
	timeout  int64
	callback func(context.Context) error
}

func (n *timeoutNode) isCanceled() bool {
	return atomic.LoadInt32(&n.canceled) != 0
}

func (n *timeoutNode) Cancel() {
	atomic.StoreInt32(&n.canceled, 1)
}

type timingWheel struct {
	group     *errgroup.Group
	startTime int64
	// the duration between tick
	tickDuration time.Duration
	// the size of the wheel
	ticksPerWheel int
	mask          int64
	buckets       []*bucket
	finished      bool
	stopChan      chan struct{}
}

func (w *timingWheel) runInLoop() {
	mask := w.mask
	var tick int64 = 0
	var nextTick time.Duration
	var timeout int64

	t := time.NewTimer(w.tickDuration)

LOOP:
	for !w.finished {
		select {
		case <-t.C:

			timeout = (tick + 1) * int64(w.tickDuration)
			w.buckets[tick&mask].calls(timeout)

			nextTick = time.Duration(timeout - time.Now().UnixNano() + w.startTime)
			if nextTick < 0 {
				nextTick = 0
			}
			t.Reset(nextTick)

		case <-w.stopChan:
			break LOOP
		}
		tick++
	}
	t.Stop()
	close(w.stopChan)
}

func (w *timingWheel) PostDelayed(ctx context.Context, callback func(context.Context) error, delay time.Duration) Future {
	node := &timeoutNode{}
	node.callback = callback
	return w.post(ctx, node, time.Now().Add(delay).UnixNano())
}

func (w *timingWheel) PostAtTime(ctx context.Context, callback func(context.Context) error, nanotime int64) Future {
	node := &timeoutNode{}
	node.callback = callback
	return w.post(ctx, node, nanotime)
}

func (w *timingWheel) post(ctx context.Context, node *timeoutNode, nanotime int64) Future {
	timeout := nanotime - w.startTime
	if timeout < 0 {
		return emptyFuture
	}
	p := (timeout / int64(w.tickDuration)) & w.mask
	node.ctx = ctx
	node.timeout = timeout
	var token = make([]byte, _TokenSize)
	binary.BigEndian.PutUint16(token[0:2], uint16(p))
	io.ReadFull(rand.Reader, token[2:])
	node.token = token

	b := w.buckets[int(p)]
	b.mutex.Lock()
	b.add(node)
	b.mutex.Unlock()
	return &future{p: unsafe.Pointer(&node), token: node.token}
}

func (w *timingWheel) Stop() {
	if w.finished {
		return
	}
	w.finished = true
	select {
	case w.stopChan <- struct{}{}:
	default:
	}
}

func (w *timingWheel) Remove(token []byte) {
	if len(token) != _TokenSize {
		return
	}
	p := binary.BigEndian.Uint16(token)
	b := w.buckets[p]
	var find *timeoutNode
	b.mutex.RLock()
	for e := b.nodes.Front(); e != nil; e = e.Next() {
		node := e.Value.(*timeoutNode)
		if !node.isCanceled() && bytes.Equal(node.token, token) {
			find = node
			break
		}
	}
	b.mutex.Unlock()

	if find != nil {
		find.Cancel()
	}
}

func WithToken(ctx context.Context, token interface{}) context.Context {
	return context.WithValue(ctx, _ContextKeyToken, token)
}

func Token(ctx context.Context) interface{} {
	return ctx.Value(_ContextKeyToken)
}
