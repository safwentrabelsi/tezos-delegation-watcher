package tzkt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dipdup-net/go-lib/tzkt/data"
	"github.com/dipdup-net/go-lib/tzkt/events"
	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"

	"github.com/sirupsen/logrus"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type WebSocketClient interface {
	Connect(ctx context.Context) error
	SubscribeToHead() error
	Listen() <-chan events.Message
	Close() error
}
type Tzkt struct {
	url           string
	client        HttpClient
	wsClient      WebSocketClient
	retryAttempts int
}

type TzktInterface interface {
	GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- *types.ChanMsg) error
	SubscribeToHead(ctx context.Context, dataChan chan<- *types.ChanMsg, currentHead chan<- uint64, errorChan chan<- error)
}

var log = logrus.WithField("module", "tzktClient")

func NewClient(cfg *config.TzktConfig) *Tzkt {
	return &Tzkt{
		url: cfg.GetURL(),
		client: &http.Client{
			Timeout: time.Duration(cfg.GetTimeout()) * time.Second,
		},
		wsClient:      events.NewTzKT(fmt.Sprintf("%s/v1/ws", cfg.GetURL())),
		retryAttempts: cfg.GetRetryAttempts(),
	}

}

func (t *Tzkt) GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- *types.ChanMsg) error {
	url := fmt.Sprintf("%s/v1/operations/delegations?level=%d", t.url, level)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("Creating request failed: %v", err)
	}

	resp, err := t.executeRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("Executing request failed: %v", err)
	}
	defer resp.Body.Close()

	var delegationsResponse []types.FetchedDelegation
	err = json.NewDecoder(resp.Body).Decode(&delegationsResponse)
	if err != nil {
		return fmt.Errorf("Decoding response failed: %v", err)
	}

	if len(delegationsResponse) > 0 {
		log.Tracef("Sending %d delegations to channel", len(delegationsResponse))
		dataChan <- &types.ChanMsg{
			Level: level,
			Reorg: false,
			Data:  delegationsResponse,
		}
	}

	return nil
}

func (t *Tzkt) SubscribeToHead(ctx context.Context, dataChan chan<- *types.ChanMsg, currentHead chan<- uint64, errorChan chan<- error) {
	log.Debug("Subscribing to TzKT WebSocket for new heads")

	if err := t.wsClient.Connect(ctx); err != nil {
		errorChan <- fmt.Errorf("couldn't connect to tzkt ws: %v", err)
		return
	}
	defer t.wsClient.Close()

	if err := t.wsClient.SubscribeToHead(); err != nil {
		log.Errorf("WebSocket subscription failed: %v", err)
		errorChan <- fmt.Errorf("couldn't subscribe to tzkt head: %v", err)
		return
	}

	messageQueue := make(chan events.Message, 100)

	// Asynchronous reception and synchronous processing for ws
	go func() {
		for msg := range t.wsClient.Listen() {
			log.Tracef("Received message: %v", msg)
			messageQueue <- msg
		}
		close(messageQueue)
	}()

	var initHead uint64
	for msg := range messageQueue {
		log.Tracef("Processing message: %v", msg)
		switch msg.Type {
		case events.MessageTypeState:
			// this is the first message received from the ws
			// we use this state as the limit between blocks that will be processed by the ws
			// and blocks that will be processed by the getPastDelegations function
			currentHead <- msg.State
			initHead = msg.State
		case events.MessageTypeData:
			if msg.Channel == events.ChannelHead {
				head, ok := msg.Body.(data.Head)
				if !ok {
					errorChan <- fmt.Errorf("unexpected type %T for head message", msg.Body)
					return
				}
				if head.Level > initHead {
					log.Infof("Fetching delegations for new head level: %d", head.Level)
					err := t.GetDelegationsByLevel(ctx, head.Level, dataChan)
					if err != nil {
						errorChan <- fmt.Errorf("error fetching delegations: %v", err)
						return
					}
				}
			}
		case events.MessageTypeReorg:
			log.Debugf("Reorg detected, processing reorg for level %d", msg.State)
			dataChan <- &types.ChanMsg{
				Level: msg.State,
				Reorg: true,
			}
		}
	}
}

func (t *Tzkt) executeRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	log.Tracef("Executing HTTP request to %s", req.URL)
	return retry.DoWithData(
		func() (*http.Response, error) {
			req = req.WithContext(ctx)
			resp, err := t.client.Do(req)
			if err != nil {

				return nil, fmt.Errorf("HTTP request failed: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("non-200 status code: %v", resp.StatusCode)
			}

			return resp, nil
		},
		retry.Context(ctx),
		retry.Attempts(uint(t.retryAttempts)),
		retry.OnRetry(func(n uint, err error) {
			log.Errorf("Retry %d for error: %v", n+1, err)
		}),
	)
}
