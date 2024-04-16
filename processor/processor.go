package processor

import (
	"context"

	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	log "github.com/sirupsen/logrus"
)

type processor struct {
	store store.Storer
}

func NewProcessor(store store.Storer) *processor {
	return &processor{
		store: store,
	}
}

func (p *processor) Run(ctx context.Context, dataChannel <-chan []types.GetDelegationsResponse) {

	for {
		select {
		case <-ctx.Done():
			return
		case delegations := <-dataChannel:
			log.Infof("new delegations %+v", delegations)
			if err := p.store.SaveDelegations(delegations); err != nil {
				log.Error("Failed to save delegations: ", err)
			}

		}
	}
}
