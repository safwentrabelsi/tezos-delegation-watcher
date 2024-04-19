package poller

import (
	"context"
	"fmt"

	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/safwentrabelsi/tezos-delegation-watcher/tzkt"
	"github.com/sirupsen/logrus"
)

type Poller struct {
	tzkt      tzkt.TzktInterface
	dataChan  chan<- *types.ChanMsg
	store     store.Storer
	cfg       *config.PollerConfig
	errorChan chan<- error
}

var log = logrus.WithField("module", "poller")

func NewPoller(tzkt tzkt.TzktInterface, dataChan chan<- *types.ChanMsg, store store.Storer, cfg *config.PollerConfig, errorChan chan<- error) Poller {
	return Poller{
		tzkt:      tzkt,
		dataChan:  dataChan,
		store:     store,
		cfg:       cfg,
		errorChan: errorChan,
	}
}

func (p *Poller) Run(ctx context.Context) {
	dbLevel, err := p.store.GetCurrentLevel(ctx)
	if err != nil {
		log.Errorf("Error getting current database level: %v", err)
		return
	}
	log.Debugf("Database level retrieved: %d", dbLevel)

	currentHead := make(chan uint64)
	go p.tzkt.SubscribeToHead(ctx, p.dataChan, currentHead, p.errorChan)
	for {
		select {
		case headLevel := <-currentHead:
			log.Infof("Received new head level: %d", headLevel)
			startLevel := max(dbLevel+1, p.cfg.GetStartLevel())

			if headLevel > dbLevel {
				log.Debugf("Fetching past delegations from level %d to %d", startLevel, headLevel)
				if err := p.getPastDelegations(ctx, startLevel, headLevel); err != nil {
					p.errorChan <- fmt.Errorf("Error fetching past delegations: %v", err)
				} else {
					log.Infof("Past delegations successfully fetched and processed from level %d to %d", startLevel, headLevel)
				}
			}
		case <-ctx.Done():
			log.Info("Poller shutdown initiated, stopping operations")
			return
		}
	}
}

func (p *Poller) getPastDelegations(ctx context.Context, startLevel, endLevel uint64) error {
	for i := startLevel; i <= endLevel; i++ {
		log.Debugf("Fetching delegations for level %d", i)
		err := p.tzkt.GetDelegationsByLevel(ctx, i, p.dataChan)
		if err != nil {
			log.Errorf("Failed fetching delegations for level %d: %v", i, err)
			return fmt.Errorf("Error fetching delegations for level %d: %v", i, err)
		}
	}
	log.Infof("Delegations fetched for levels %d to %d", startLevel, endLevel)
	return nil
}
