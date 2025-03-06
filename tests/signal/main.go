package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	appDone := make(chan struct{})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		for {
			select {
			case <-appDone:
				fmt.Println(`{"severity":"INFO","timestamp":"2022-04-21T14:50:18.382974069-04:00","logger":"l","message":"Done"}`)
				return
			case <-time.After(3 * time.Second):
				fmt.Println(`{"severity":"INFO","timestamp":"2022-04-21T14:50:18.382974069-04:00","logger":"l","message":"Running"}`)
			}
		}
	}()

	<-ctx.Done()
	fmt.Println(`{"severity":"INFO","timestamp":"2022-04-21T14:50:18.382974069-04:00","logger":"l","message":"Termination signal received"}`)

	close(appDone)
	time.Sleep(3 * time.Second)

	fmt.Println(`{"severity":"INFO","timestamp":"2022-04-21T14:50:18.382974069-04:00","logger":"l","message":"Exiting"}`)
}
