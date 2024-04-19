package poller

import (
	"context"

	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/safwentrabelsi/tezos-delegation-watcher/tzkt"
	log "github.com/sirupsen/logrus"
)

type Poller struct {
	tzkt     tzkt.TzktInterface
	dataChan chan<- *types.ChanMsg
	store    store.Storer
	cfg      *config.PollerConfig
}

func NewPoller(tzkt tzkt.TzktInterface, dataChan chan<- *types.ChanMsg, store store.Storer, cfg *config.PollerConfig) Poller {
	return Poller{
		tzkt:     tzkt,
		dataChan: dataChan,
		store:    store,
		cfg:      cfg,
	}
}
func (p *Poller) Run(ctx context.Context) {
	dbLevel, err := p.store.GetCurrentLevel(ctx)
	if err != nil {
		log.Errorf("Error getting current level: %v", err)
		return
	}

	currentHead := make(chan uint64)
	errorChan := make(chan error)
	go p.tzkt.SubscribeToHead(ctx, p.dataChan, currentHead, errorChan)

	select {
	case headLevel := <-currentHead:
		log.Infof("Received head level: %d", headLevel)
		startLevel := max(dbLevel+1, p.cfg.GetStartLevel())

		if headLevel > dbLevel {
			if err := p.getPastDelegations(ctx, startLevel, headLevel); err != nil {
				log.Errorf("Error fetching past delegations from level %d to %d: %v", dbLevel+1, headLevel, err)
			}
		}
	case err := <-errorChan:
		log.Errorf("Error from WebSocket subscription: %v", err)
		// Decide how to handle the error - retry, alert, or stop
		return
	case <-ctx.Done():
		log.Info("Context cancelled, stopping Poller")
		return
	}
}
func (p *Poller) getPastDelegations(ctx context.Context, startLevel, endLevel uint64) error {
	for i := startLevel; i <= endLevel; i++ {
		err := p.tzkt.GetDelegationsByLevel(ctx, i, p.dataChan)
		if err != nil {
			log.Errorf("Error fetching delegations for level %d: %v", i, err)
			continue // Maybe retry fetching this level
		}
	}
	return nil
}
