package tzkt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
type Tzkt struct {
	URL    string
	Client HttpClient
}

type TzktInterface interface {
	GetDelegations(ctx context.Context, dataChan chan<- []types.GetDelegationsResponse) error
	GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- []types.GetDelegationsResponse) error
	GetHead(ctx context.Context) (uint64, error)
}

func NewClient(cfg *config.TzktConfig) *Tzkt {
	return &Tzkt{
		URL: cfg.GetURL(),
		Client: &http.Client{
			Timeout: time.Duration(cfg.GetTimeout()) * time.Second,
		}}
}

func (t *Tzkt) GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- []types.GetDelegationsResponse) error {

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
		dataChan <- delegationsResponse
	}
	return nil
}

func (t *Tzkt) GetDelegations(ctx context.Context, dataChan chan<- []types.GetDelegationsResponse) error {
	var i int
	// Ensure it's non-empty for the first loop entry
	delegationsResponse := make([]types.GetDelegationsResponse, 1)
	for len(delegationsResponse) > 0 {
		// get offset from config
		url := fmt.Sprintf("%s/v1/operations/delegations?limit=%d&offset=%d", t.URL, 10000, i*10000)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := t.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Reset before decoding to avoid mixing old data
		delegationsResponse = nil
		err = json.NewDecoder(resp.Body).Decode(&delegationsResponse)
		if err != nil {
			return err
		}

		if len(delegationsResponse) > 0 {
			dataChan <- delegationsResponse
			i++ // Increment to fetch next set in pagination
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
