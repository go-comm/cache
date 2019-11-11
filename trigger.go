package cache

const (
	EventSourceDelete uint8 = iota
	EventSourcePut
)

type Event struct {
	Source uint8
	Key    []byte
}

type DeleteTrigger interface {
	OnEvent(*Event)
}
