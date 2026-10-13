package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	godotenv.Load()

	numberBots := flag.Int("bots", 100, "Number of bots to spawn")
	rampUp := flag.Bool("ramp-up", true, "Enable ramp-up mode for bot spawning")
	rampDuration := flag.Int("ramp-duration", 60, "Duration in seconds for ramp-up mode")
	sustainDuration := flag.Int("sustain-duration", 120, "Duration in seconds to sustain max load")
	rampDown := flag.Bool("ramp-down", true, "Enable ramp-down mode")
	flag.Parse()

	totalDuration := *rampDuration + *sustainDuration + *rampDuration
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(totalDuration)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	shutdownChan := make(chan int, *numberBots)
	spawn := func(n int) {
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				runBot(ctx, &wg, shutdownChan)
			}()
		}
	}

	if *rampUp && *rampDuration > 0 {
		base := *numberBots / *rampDuration
		rem := *numberBots % *rampDuration

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for i := 0; i < *rampDuration; i++ {
			toSpawn := base
			if i < rem {
				toSpawn++
			}
			spawn(toSpawn)
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	} else {
		spawn(*numberBots)
	}

	time.Sleep(time.Duration(*sustainDuration) * time.Second)

	if *rampDown && *rampDuration > 0 {
		base := *numberBots / *rampDuration
		rem := *numberBots % *rampDuration

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for i := 0; i < *rampDuration; i++ {
			toStop := base
			if i < rem {
				toStop++
			}
			for j := 0; j < toStop; j++ {
				select {
				case shutdownChan <- 1:
				default:
				}
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Wait()
}

type ClientMsg struct {
	PlayerID  string `json:"playerID"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Direction string `json:"direction"`
}

func runBot(ctx context.Context, wg *sync.WaitGroup, shutdownChan <-chan int) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		return
	default:
		host := os.Getenv("SERVER_HOST")
		if host == "" {
			host = "127.0.0.1"
		}

		port := os.Getenv("SERVER_PORT")
		if port == "" {
			port = "4200"
		}

		addr := net.JoinHostPort(host, port)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Printf("bot failed to connect: %v", err)
			return
		}
		defer conn.Close()

		encoder := json.NewEncoder(conn)

		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-shutdownChan:
				return
			case <-t.C:
				direction := getRandomDirection()
				msg := ClientMsg{Type: "move", Direction: direction}
				if err := encoder.Encode(msg); err != nil {
					return
				}
			}
		}

	}
}

func getRandomDirection() string {
	directions := []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}
	return directions[rand.Intn(len(directions))]
}
