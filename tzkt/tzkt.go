package tzkt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	URL    string
	Client HttpClient
}

type TzktInterface interface {
	GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- *types.ChanMsg) error
	GetHead(ctx context.Context) (uint64, error)
	SubscribeToHead(ctx context.Context, dataChan chan<- *types.ChanMsg, currentHead chan<- uint64, errorChan chan<- error)
}

func NewClient(cfg *config.TzktConfig) *Tzkt {
	return &Tzkt{
		URL: cfg.GetURL(),
		Client: &http.Client{
			Timeout: time.Duration(cfg.GetTimeout()) * time.Second,
		}}
}

func (t *Tzkt) GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- *types.ChanMsg) error {
	url := fmt.Sprintf("%s/v1/operations/delegations?level=%d", t.URL, level)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}

	var delegationsResponse []types.GetDelegationsResponse
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

func (t *Tzkt) GetHead(ctx context.Context) (uint64, error) {
	url := fmt.Sprintf("%s/v1/blocks/count", t.URL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return 0, err
	}

	var level uint64
	err = json.NewDecoder(resp.Body).Decode(&level)
	if err != nil {
		return 0, err
	}
	return level, nil
}

func (t *Tzkt) SubscribeToHead(ctx context.Context, dataChan chan<- *types.ChanMsg, currentHead chan<- uint64, errorChan chan<- error) {
	tzkt := events.NewTzKT(fmt.Sprintf("%s/v1/ws", t.URL))
	if err := tzkt.Connect(ctx); err != nil {
		errorChan <- fmt.Errorf("couldn't connect to tzkt ws: %v", err)
		return
	}
	defer tzkt.Close()

	if err := tzkt.SubscribeToHead(); err != nil {
		errorChan <- fmt.Errorf("couldn't subscribe to tzkt head: %v", err)
		return
	}

	var initHead uint64
	for msg := range tzkt.Listen() {
		switch msg.Type {
		// This is he first message received by the ws
		case events.MessageTypeState:
			// Set the limit of the init fetch so no block would be missed or fetched twice
			// In the documentation for a head subscription State contains level (int) of the last processed head.
			// So we will be handling starting from the next head provided by the ws
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
					if err := t.GetDelegationsByLevel(ctx, head.Level, dataChan); err != nil {
						logrus.Error("error fetching delegations: ", err)
					}
				}
			}
		case events.MessageTypeReorg:
			dataChan <- &types.ChanMsg{
				Level: msg.State,
				Reorg: true,
			}

		}
		logrus.Info("Received message: ", msg)
	}
}
