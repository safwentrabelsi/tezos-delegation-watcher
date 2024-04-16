package main

import (
	"context"
	"time"

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
	dataChannel := make(chan []types.GetDelegationsResponse, 100)

	tzktClient := tzkt.NewClient(cfg.Tzkt)
	delegationPoller := poller.NewPoller(tzktClient, 15*time.Second, dataChannel, store, cfg.Tzkt.GetStartLevel())
	go delegationPoller.Run(context.TODO())
	delegationProcessor := processor.NewProcessor(store)
	go delegationProcessor.Run(context.TODO(), dataChannel)
	server := api.NewAPIServer(cfg.Server, store)
	server.Run()

}
