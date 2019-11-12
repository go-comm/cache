package cache

type Cache interface {
	Get(k []byte) (interface{}, error)
	Put(k []byte, v interface{}) error
	PutEx(k []byte, v interface{}, sec int64) error
	Del(k []byte) error
	List([]*Entry) []*Entry
	TTL(k []byte) int64
	Expire(k []byte, ex int64)
}
