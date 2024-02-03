package leonai

import (
	"context"
	"log"
	"time"
)

// Server serves the leonai server.
func Serve(ctx context.Context, port int) error {
	log.Printf("server listening on port %d\n", port)
	<-ctx.Done()
	return nil
}

// Run runs the leonai process.
func Run(ctx context.Context) error {
	log.Println("running")
	defer log.Println("finished")
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
	}
	return nil
}
