package processor

import (
	"context"
	"fmt"

	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Run(ctx context.Context, dataChannel <-chan *types.ChanMsg)
}

type processor struct {
	store store.Storer
}

func NewProcessor(store store.Storer) Processor {
	return &processor{
		store: store,
	}
}

func (p *processor) Run(ctx context.Context, dataChannel <-chan *types.ChanMsg) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-dataChannel:
			if !msg.Reorg {
				err := p.processDelegations(ctx, msg.Data)
				if err != nil {
					logrus.Errorf("Failed to process delegations: %v", err)
				}
			} else {
				err := p.processReorg(ctx, msg.Level)
				if err != nil {
					logrus.Errorf("Failed to process reorg: %v", err)
				}
			}

		}
	}
}

func (p *processor) processDelegations(ctx context.Context, delegations []types.GetDelegationsResponse) error {
	logrus.Infof("Processing %d delegations", len(delegations))
	logrus.Infof("Processing %+v", delegations)

	err := p.store.SaveDelegations(ctx, delegations)
	if err != nil {
		return fmt.Errorf("failed to save delegations: %w", err)
	}

	return nil
}

func (p *processor) processReorg(ctx context.Context, level uint64) error {
	logrus.Infof("Processing reorg of from block %d ", level)

	err := p.store.DeleteDelegationsFromLevel(ctx, level)
	if err != nil {
		return fmt.Errorf("failed to delete delegations: %w", err)
	}

	return nil
}
