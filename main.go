package main

import (
	"context"

	"github.com/safwentrabelsi/tezos-delegation-watcher/api"
	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/poller"
	"github.com/safwentrabelsi/tezos-delegation-watcher/processor"
	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/safwentrabelsi/tezos-delegation-watcher/tzkt"
	log "github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		return
	}

	// log configuration TODO create logger package
	logLevel, err := log.ParseLevel(cfg.Log.GetLevel())
	if err != nil {
		log.Fatal("Invalid log level in the config: ", err)
	}
	log.SetLevel(logLevel)

	store, err := store.NewPostgresStore(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to initialize Postgres store: %v", err)
	}
	dataChannel := make(chan *types.ChanMsg, 100)
	ctx := context.Background()
	tzktClient := tzkt.NewClient(cfg.Tzkt)
	delegationPoller := poller.NewPoller(tzktClient, dataChannel, store, cfg.Poller)
	delegationProcessor := processor.NewProcessor(store, dataChannel)
	go delegationPoller.Run(ctx)
	go delegationProcessor.Run(ctx)
	server := api.NewAPIServer(cfg.Server, store)
	server.Run()

}
