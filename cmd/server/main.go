package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LealKevin/terminus/internal/app"
	"github.com/LealKevin/terminus/internal/infra/server"
	"github.com/LealKevin/terminus/internal/infra/store"
)

func main() {
	worldMemoryStore := store.NewWorldMemoryStore()
	playerMemoryStore := store.NewPlayerMemoryStore()
	mobMemoryStore := store.NewMobMemoryStore()
	handler := app.NewHandler(worldMemoryStore, playerMemoryStore, mobMemoryStore)
	server := server.NewServer(":4200", handler)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go StartGameLoop(ctx, handler)
	server.Start(ctx)
}

func StartGameLoop(ctx context.Context, h *app.Handler) {
	ticker := time.NewTicker(time.Second * 1 / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			mobsCount := h.Mobs.CountMobsInWorld("world1")
			if mobsCount < 5 {
				for i := 0; i < 5-mobsCount; i++ {
					h.HandleMobSpawn(ctx, "world1", "Goblin", "Goblin")
				}
			}

			for _, mob := range h.Mobs.GetMobsByWorld("world1") {
				if rand.Float32() < 0.5 {
					h.HandleMobMove(ctx, mob.ID)
				}
			}

			err := h.BroadcastMobsUpdate("world1")
			if err != nil {
				log.Printf("Failed to broadcast mob update: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
