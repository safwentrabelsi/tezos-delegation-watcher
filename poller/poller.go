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
	GetFetchOld() bool
}

type poller struct {
	tzkt      tzkt.TzktInterface
	dataChan  chan<- *types.ChanMsg
	store     storeInterface
	cfg       configInterface
	errorChan chan<- error
}

var log = logrus.WithField("module", "poller")

// NewPoller creates a new Poller instance with the necessary dependencies.
func NewPoller(tzkt tzkt.TzktInterface, dataChan chan<- *types.ChanMsg, store storeInterface, cfg configInterface, errorChan chan<- error) *poller {
	return &poller{
		tzkt:      tzkt,
		dataChan:  dataChan,
		store:     store,
		cfg:       cfg,
		errorChan: errorChan,
	}
}

// Run starts the polling process.
func (p *poller) Run(ctx context.Context) {

	log.Info("Starting the poller...")

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
				// if fetchOld is true in config we proceed to fetch old delegations from the startLevel or the current db level.
				if p.cfg.GetFetchOld() {
					log.Info("Fetching old delegations is activated")
					dbLevel, err := p.store.GetCurrentLevel(ctx)
					if err != nil {
						p.errorChan <- fmt.Errorf("Error getting current database level: %v", err)
					}
					log.Infof("Database level retrieved: %d", dbLevel)
					log.Infof("Received chain current head level: %d", headLevel)

					startLevel := max(dbLevel+1, p.cfg.GetStartLevel())
					if headLevel > dbLevel {
						log.Debugf("Fetching past delegations from level %d to %d", startLevel, headLevel)
						if err := p.getPastDelegations(ctx, startLevel, headLevel); err != nil {
							p.errorChan <- fmt.Errorf("Error fetching past delegations: %v", err)
						}
						log.Infof("Past delegations successfully fetched and processed from level %d to %d", startLevel, headLevel)
					}
				} else {
					log.Info("Fetching old delegations is deactivated, Only new delegations will be processed")
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
		// The retry logic is here  and not in the tzkt module  because we should be aware in case of block delta when the connection was closed
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

// getPastDelegations fetches delegations from the provided start level to the end level.
func (p *poller) getPastDelegations(ctx context.Context, startLevel, endLevel uint64) error {
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
