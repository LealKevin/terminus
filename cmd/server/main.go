package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LealKevin/terminus/internal/app"
	"github.com/LealKevin/terminus/internal/infra/db"
	"github.com/LealKevin/terminus/internal/infra/server"
	"github.com/LealKevin/terminus/internal/infra/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Prometheus
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:    ":2112",
		Handler: metricsMux,
	}

	go func() {
		log.Printf("metrics listening on :2112")
		if err := metricsSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("metrics server error: %v", err)
		}
	}()

	database, err := db.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	queries := db.New(database.Conn)
	stores := store.NewPgBundle(queries)
	handler := app.NewHandler(stores)
	server := server.NewServer(":4200", handler)

	go StartGameLoop(ctx, handler)
	go func() {
		log.Println("app listening on :4200")
		server.Start(ctx)
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = metricsSrv.Shutdown(shutdownCtx)
}

func StartGameLoop(ctx context.Context, h *app.Handler) {
	ticker := time.NewTicker(time.Second * 1 / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			mobsCount, err := h.Mobs.CountMobsInWorld(ctx, "123e4567-e89b-12d3-a456-426614174000")
			if err != nil {
				log.Printf("Failed to count mobs: %v", err)
				continue
			}
			if mobsCount < 5 {
				for i := 0; i < 5-mobsCount; i++ {
					if _, err := h.HandleMobSpawn(ctx, "123e4567-e89b-12d3-a456-426614174000", "Goblin", "Goblin"); err != nil {
						log.Printf("Failed to spawn mob: %v", err)
					}
				}
			}

			mobs, err := h.Mobs.GetMobsByWorld(ctx, "123e4567-e89b-12d3-a456-426614174000")
			if err != nil {
				log.Printf("Failed to list mobs: %v", err)
				continue
			}
			for _, mob := range mobs {
				if rand.Float32() < 0.5 {
					if err := h.HandleMobMove(ctx, mob.ID); err != nil {
						log.Printf("Failed to move mob %s: %v", mob.ID, err)
					}
				}
			}

			if err := h.BroadcastMobsUpdate(ctx, "123e4567-e89b-12d3-a456-426614174000"); err != nil {
				log.Printf("Failed to broadcast mob update: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
