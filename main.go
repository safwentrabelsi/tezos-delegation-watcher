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
	"github.com/safwentrabelsi/tezos-delegation-watcher/utils"
	log "github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	logLevel, err := log.ParseLevel(cfg.Log.GetLevel())
	if err != nil {
		log.Fatal("Invalid log level in the config: ", err)
	}
	log.SetLevel(logLevel)

	store, err := store.NewPostgresStore(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to initialize Postgres store: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errorChan := make(chan error, 2)
	defer close(errorChan)

	dataChannel := make(chan *types.ChanMsg, 100)
	defer close(dataChannel)

	tzktClient := tzkt.NewClient(cfg.Tzkt)

	delegationPoller := poller.NewPoller(tzktClient, dataChannel, store, cfg.Poller, errorChan)
	delegationProcessor := processor.NewProcessor(store, dataChannel, errorChan)

	go delegationPoller.Run(ctx)
	go delegationProcessor.Run(ctx)
	go utils.HandleErrors(ctx, cancel, errorChan)

	server := api.NewAPIServer(cfg.Server, store)
	server.Run()
}
