package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

type ConnHandler interface {
	HandleConnection(context.Context, net.Conn)
}

type Server struct {
	Port    string
	Handler ConnHandler
}

func NewServer(port string, handler ConnHandler) *Server {
	return &Server{
		Port:    port,
		Handler: handler,
	}
}

func (s *Server) Start(ctx context.Context) {
	ln, err := net.Listen("tcp", s.Port)
	if err != nil {
		log.Fatalf("unable to start server: %v", err)
	}
	defer ln.Close()

	fmt.Printf("listener started on port %s \n", s.Port)

	var wg sync.WaitGroup
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("error accepting connection: %v", err)
				return
			}

			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				s.Handler.HandleConnection(ctx, c)
			}(conn)
		}
	}()

	<-ctx.Done()

	fmt.Println("\nShutting down the server...")
	wg.Wait()
	fmt.Println("Server gracefully stopped")
}
