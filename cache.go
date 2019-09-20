package cache

type Cache interface {
	Get(k []byte) (interface{}, error)
	Put(k []byte, v interface{}) error
	PutEx(k []byte, v interface{}, sec int64) error
}
