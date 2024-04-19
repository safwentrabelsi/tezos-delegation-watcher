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
type Tzkt struct {
	url    string
	client HttpClient
}

type TzktInterface interface {
	GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- *types.ChanMsg) error
	SubscribeToHead(ctx context.Context, dataChan chan<- *types.ChanMsg, currentHead chan<- uint64, errorChan chan<- error)
}

func NewClient(cfg *config.TzktConfig) *Tzkt {
	return &Tzkt{
		url: cfg.GetURL(),
		client: &http.Client{
			Timeout: time.Duration(cfg.GetTimeout()) * time.Second,
		}}

}

func (t *Tzkt) GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- *types.ChanMsg) error {
	url := fmt.Sprintf("%s/v1/operations/delegations?level=%d", t.url, level)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := t.executeRequest(ctx, req)
	if err != nil {
		return err
	}

	var delegationsResponse []types.FetchedDelegation
	err = json.NewDecoder(resp.Body).Decode(&delegationsResponse)
	if err != nil {
		return err
	}

	if len(delegationsResponse) > 0 {
		dataChan <- &types.ChanMsg{
			Level: level,
			Reorg: false,
			Data:  delegationsResponse,
		}
	}

	return nil
}

func (t *Tzkt) SubscribeToHead(ctx context.Context, dataChan chan<- *types.ChanMsg, currentHead chan<- uint64, errorChan chan<- error) {
	tzkt := events.NewTzKT(fmt.Sprintf("%s/v1/ws", t.url))
	if err := tzkt.Connect(ctx); err != nil {
		errorChan <- fmt.Errorf("couldn't connect to tzkt ws: %v", err)
		return
	}
	defer tzkt.Close()

	if err := tzkt.SubscribeToHead(); err != nil {
		errorChan <- fmt.Errorf("couldn't subscribe to tzkt head: %v", err)
		return
	}

	messageQueue := make(chan events.Message, 100)

	// Goroutine to listen to WebSocket and queue messages
	go func() {
		for msg := range tzkt.Listen() {
			messageQueue <- msg
		}
		close(messageQueue)
	}()

	var initHead uint64
	// Process messages from the queue
	for msg := range messageQueue {
		switch msg.Type {
		case events.MessageTypeState:
			currentHead <- msg.State
			initHead = msg.State
		case events.MessageTypeData:
			if msg.Channel == events.ChannelHead {
				head, ok := msg.Body.(data.Head)
				if !ok {
					errorChan <- fmt.Errorf("unexpected type %T for head message", msg.Body)
					continue
				}
				if head.Level > initHead {
					err := t.GetDelegationsByLevel(ctx, head.Level, dataChan)
					if err != nil {
						errorChan <- fmt.Errorf("error fetching delegations: %v", err)
					}
				}
			}
		case events.MessageTypeReorg:
			dataChan <- &types.ChanMsg{
				Level: msg.State,
				Reorg: true,
			}
		}
		logrus.Trace("Processed message: ", msg)
	}
}

func (t *Tzkt) executeRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	return retry.DoWithData(
		func() (*http.Response, error) {
			req = req.WithContext(ctx)
			resp, err := t.client.Do(req)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("non 200 status code: %v", resp.StatusCode)
			}

			return resp, nil
		},
		retry.Context(ctx),
		retry.Attempts(3),
		retry.OnRetry(func(n uint, err error) {
			logrus.Errorf("failed to fetch data err: %v, retrying...", err)
		}),
	)
}
