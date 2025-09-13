package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type server struct {
	port string
}

type client struct {
	port string
}

func main() {
	port := ":4200"
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("unable to start server")
	}

	defer l.Close()
	fmt.Printf("listenner started on port %s \n", port)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Printf("error accepting connection: %v", err)
				continue
			}
			wg.Add(1)
			go handleConnection(conn, &wg)
		}
	}()

	<-stop
	fmt.Println("\nShutting down the server...")
	l.Close()

	wg.Wait()
	fmt.Print("")
}

func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()
	fmt.Printf("New connection from: %v \n", conn.RemoteAddr().String())
	for {
		reader := bufio.NewReader(conn)

		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Printf("error reading from connection: %v", err)
			return
		}
		conn.Write(line)
	}
}
