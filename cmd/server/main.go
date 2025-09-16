package main

import (
	"github.com/LealKevin/terminus/internal/app"
	"github.com/LealKevin/terminus/internal/infra/server"
	"github.com/LealKevin/terminus/internal/infra/store"
)

func main() {
	worldMemoryStore := store.NewWorldMemoryStore()
	playerMemoryStore := store.NewPlayerMemoryStore()
	handler := app.NewHandler(worldMemoryStore, playerMemoryStore)
	server := server.NewServer(":4200", handler)
	server.Start()
}
