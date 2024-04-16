package poller

import (
	"context"
	"time"

	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/safwentrabelsi/tezos-delegation-watcher/tzkt"
	log "github.com/sirupsen/logrus"
)

type Poller struct {
	tzkt       tzkt.TzktInterface
	interval   time.Duration
	dataChan   chan<- []types.GetDelegationsResponse
	store      store.Storer
	startLevel uint64
}

func NewPoller(tzkt tzkt.TzktInterface, interval time.Duration, dataChan chan<- []types.GetDelegationsResponse, store store.Storer, startLevel uint64) Poller {
	return Poller{
		tzkt:       tzkt,
		interval:   interval,
		dataChan:   dataChan,
		store:      store,
		startLevel: startLevel,
	}
}

func (p *Poller) Run(ctx context.Context) {
	dbLevel, err := p.store.GetCurrentLevel(ctx)

	log.Info(dbLevel)
	if err != nil {
		log.Errorf("Error getting current level: %v", err)
		return
	}

	var startLevel uint64
	if dbLevel < p.startLevel {
		startLevel = p.startLevel
	} else {
		startLevel = dbLevel + 1
	}

	headLevel, err := p.tzkt.GetHead(ctx)
	log.Info(headLevel)
	if err != nil {
		log.Errorf("Error getting head level from tzkt: %v", err)
		return
	}
	go func() {
		// Regular polling
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				latestLevel, err := p.tzkt.GetHead(ctx)
				if err != nil {
					log.Errorf("Error fetching current head from tzkt: %v", err)
					continue
				}

				if latestLevel > headLevel {
					log.Info("new level")
					err := p.tzkt.GetDelegationsByLevel(ctx, latestLevel, p.dataChan)
					if err != nil {
						log.Errorf("Error fetching delegations for level %d: %v", latestLevel, err)
						continue
					}
					headLevel = latestLevel
				}
			}
		}
	}()

	// Fetch delegations for any missed blocks if there's a deficit
	if headLevel > startLevel {
		if err := p.getPastDelegations(ctx, startLevel, headLevel); err != nil {
			log.Errorf("Error fetching past delegations from level %d to %d: %v", startLevel+1, headLevel, err)
			return
		}
	}

}

func (p *Poller) getPastDelegations(ctx context.Context, startLevel, endLevel uint64) error {
	for i := startLevel + 1; i <= endLevel; i++ {
		err := p.tzkt.GetDelegationsByLevel(ctx, i, p.dataChan)
		if err != nil {
			log.Errorf("Error fetching delegations for level %d: %v", i, err)
			continue // Maybe retry fetching this level
		}
	}
	return nil
}
