package main

import (
	"context"
	"log"
	"time"

	"github.com/go-comm/cache/internal/timingwheel"
)

func main() {
	tw := timingwheel.New(time.Millisecond*100, 256)

	var fs []timingwheel.Future

	callback := func(ctx context.Context) error {
		log.Println("done", ctx.Value("Index"), ctx.Value("Delay"))
		return nil
	}

	for i := 0; i < 100; i++ {
		delay := i * 200
		ctx := context.Background()
		ctx = context.WithValue(ctx, "Index", i)
		ctx = context.WithValue(ctx, "Delay", delay)
		f := tw.PostDelayed(ctx, callback, time.Duration(delay)*time.Millisecond)
		fs = append(fs, f)
	}

	time.Sleep(time.Second * 5)

	log.Println("Cancel all.")

	for _, f := range fs {
		f.Cancel()
	}

	time.Sleep(time.Second * 1000)

}
