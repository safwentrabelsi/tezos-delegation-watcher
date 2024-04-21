package processor

import (
	"context"
	"fmt"

	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Run(ctx context.Context)
}

type processor struct {
	store       store.Storer
	dataChannel <-chan *types.ChanMsg
	errorChan   chan<- error
}

var log = logrus.WithField("module", "processor")

func NewProcessor(store store.Storer, dataChannel <-chan *types.ChanMsg, errorChan chan<- error) Processor {
	return &processor{
		store:       store,
		dataChannel: dataChannel,
		errorChan:   errorChan,
	}
}

func (p *processor) Run(ctx context.Context) {
	log.Info("Starting Processor")
	for {
		select {
		case <-ctx.Done():
			log.Info("Processor stopping due to context cancellation")
			return
		case msg := <-p.dataChannel:
			if msg == nil {
				continue
			}
			if !msg.Reorg {
				log.Infof("Received new delegations at level %d", msg.Level)
				err := p.processDelegations(ctx, msg.Data)
				if err != nil {
					log.WithError(err).Error("Failed to process delegations")
					p.errorChan <- err
				}
			} else {
				log.Debugf("Received reorg command for level %d", msg.Level)
				err := p.processReorg(ctx, msg.Level)
				if err != nil {
					log.WithError(err).Error("Failed to process reorg")
					p.errorChan <- err
				}
			}
		}
	}
}

func (p *processor) processDelegations(ctx context.Context, delegations []types.FetchedDelegation) error {
	if len(delegations) == 0 {
		log.Debug("No delegations to process")
		return nil
	}
	log.Infof("Processing %d delegations", len(delegations))
	err := p.store.SaveDelegations(ctx, delegations)
	if err != nil {
		log.WithError(err).Error("Failed to save delegations")
		return fmt.Errorf("failed to save delegations: %w", err)
	}
	log.Info("Delegations processed and saved successfully")
	return nil
}

func (p *processor) processReorg(ctx context.Context, level uint64) error {
	log.Infof("Processing reorganization from block level %d", level)
	err := p.store.DeleteDelegationsFromLevel(ctx, level)
	if err != nil {
		log.WithError(err).Error("Failed to delete delegations during reorg")
		return fmt.Errorf("failed to delete delegations: %w", err)
	}
	log.Info("Reorganization processed successfully")
	return nil
}
