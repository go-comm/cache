package cache

import "time"

const (
	timeOfMillisecond = int64(time.Millisecond)
)

func nowTime() int64 {
	return time.Now().UnixNano() / timeOfMillisecond
}
