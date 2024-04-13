package main

import (
	"github.com/safwentrabelsi/tezos-delegation-watcher/api"
	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	log "github.com/sirupsen/logrus"
)

func main() {
	store, err := store.NewPostgresStore()
	if err != nil {
		log.Fatalf("Failed to initialize Postgres store: %v", err)
	}

	server := api.NewAPIServer(":3000", store)
	server.Run()

}
