package processor

import (
	"context"
	"fmt"

	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Run(ctx context.Context, dataChannel <-chan []types.GetDelegationsResponse)
}

type processor struct {
	store store.Storer
}

func NewProcessor(store store.Storer) Processor {
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
			err := p.processDelegations(ctx, delegations)
			if err != nil {
				logrus.Errorf("Failed to process delegations: %v", err)
			}
		}
	}
}

func (p *processor) processDelegations(ctx context.Context, delegations []types.GetDelegationsResponse) error {
	logrus.Infof("Processing %d delegations", len(delegations))

	err := p.store.SaveDelegations(ctx, delegations)
	if err != nil {
		return fmt.Errorf("failed to save delegations: %w", err)
	}

	return nil
}
