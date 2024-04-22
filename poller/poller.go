package poller

import (
	"context"
	"fmt"
	"time"

	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/safwentrabelsi/tezos-delegation-watcher/tzkt"
	"github.com/sirupsen/logrus"
)

type storeInterface interface {
	GetCurrentLevel(ctx context.Context) (uint64, error)
}

type configInterface interface {
	GetStartLevel() uint64
	GetRetryAttempts() int
}
type Poller struct {
	tzkt      tzkt.TzktInterface
	dataChan  chan<- *types.ChanMsg
	store     storeInterface
	cfg       configInterface
	errorChan chan<- error
}

var log = logrus.WithField("module", "poller")

func NewPoller(tzkt tzkt.TzktInterface, dataChan chan<- *types.ChanMsg, store storeInterface, cfg configInterface, errorChan chan<- error) Poller {
	return Poller{
		tzkt:      tzkt,
		dataChan:  dataChan,
		store:     store,
		cfg:       cfg,
		errorChan: errorChan,
	}
}

func (p *Poller) Run(ctx context.Context) {
	// retry attempts if connection to ws failed
	attempt := 0

	connect := func() error {
		currentHead := make(chan uint64)
		defer close(currentHead)

		errChan := make(chan error, 1)
		defer close(errChan)

		go p.tzkt.SubscribeToHead(ctx, p.dataChan, currentHead, errChan)

		for {
			select {
			// if there is a problem with SubscribeToHead the connect function will return an error
			// and before this error is pushed to the main error channel we retry to connect
			case err := <-errChan:
				return err
			// to be sure there is no delta between past blocks and blocks comming from the ws
			case headLevel := <-currentHead:
				dbLevel, err := p.store.GetCurrentLevel(ctx)
				if err != nil {
					p.errorChan <- fmt.Errorf("Error getting current database level: %v", err)
				}
				log.Debugf("Database level retrieved: %d", dbLevel)
				log.Infof("Received chain current head level: %d", headLevel)

				startLevel := max(dbLevel+1, p.cfg.GetStartLevel())
				if headLevel > dbLevel {
					log.Debugf("Fetching past delegations from level %d to %d", startLevel, headLevel)
					if err := p.getPastDelegations(ctx, startLevel, headLevel); err != nil {
						p.errorChan <- fmt.Errorf("Error fetching past delegations: %v", err)
					}
					log.Infof("Past delegations successfully fetched and processed from level %d to %d", startLevel, headLevel)
				}
			case <-ctx.Done():
				log.Info("Poller shutdown initiated, stopping operations")
				return nil
			}
		}
	}

	for {
		err := connect()
		if err == nil || ctx.Err() != nil {
			// Reset attempt to zero to have the maximum retries for the next failure
			attempt = 0
			log.Debug("Stopping reconnection attempts")
			return
		}
		// The retry logic is here because and not in the tzkt module we should be aware in case of block delta when the connection was closed
		if attempt < p.cfg.GetRetryAttempts() {
			waitTime := 1 * time.Second
			log.Errorf("Attempt %d: Connection failed with error: %v. Retrying in %v...", attempt+1, err, waitTime)
			time.Sleep(waitTime)
			attempt++
		} else {
			// if we attempted max retries with no success we push the error to the main errorChan which will stop the server
			p.errorChan <- fmt.Errorf("maximum reconnection attempts reached: %v", err)
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
