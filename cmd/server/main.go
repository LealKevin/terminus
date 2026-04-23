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
	"github.com/LealKevin/terminus/internal/domain"
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

	if err := initializeWorld(ctx, handler); err != nil {
		log.Fatalf("Failed to initialize world: %v", err)
	}

	worldID := os.Getenv("WORLD_ID")
	if worldID == "" {
		worldID = "123e4567-e89b-12d3-a456-426614174000"
	}
	if err := handler.LoadMobs(ctx, worldID); err != nil {
		log.Fatalf("Failed to load mobs: %v", err)
	}

	go StartGameLoop(ctx, handler, worldID)
	go handler.SaveDirtyPlayers(ctx)
	go handler.FlushMobs(ctx)

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

func StartGameLoop(ctx context.Context, h *app.Handler, worldID string) {
	ticker := time.NewTicker(time.Second * 1 / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			mobsCount := h.MobCountInWorld(worldID)
			if mobsCount < 5 {
				for i := 0; i < 5-mobsCount; i++ {
					if _, err := h.HandleMobSpawn(ctx, worldID, "Goblin", "Goblin"); err != nil {
						log.Printf("Failed to spawn mob: %v", err)
					}
				}
			}

			for _, id := range h.MobIDsInWorld(worldID) {
				if rand.Float32() < 0.5 {
					if err := h.HandleMobMove(ctx, id); err != nil {
						log.Printf("Failed to move mob %s: %v", id, err)
					}
				}
			}

			if err := h.BroadcastMobsUpdate(ctx, worldID); err != nil {
				log.Printf("Failed to broadcast mob update: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func initializeWorld(ctx context.Context, h *app.Handler) error {
	worldID := os.Getenv("WORLD_ID")
	if worldID == "" {
		worldID = "123e4567-e89b-12d3-a456-426614174000"
	}

	_, err := h.Worlds.GetWorld(ctx, worldID)
	if err == nil {
		log.Println("World already exists")
		return nil
	}

	log.Println("Creating initial world...")
	if _, err := h.Worlds.CreateWorld(ctx, worldID, 53, 25, domain.Raw); err != nil {
		return err
	}

	log.Printf("World %s created successfully", worldID)
	return nil
}
