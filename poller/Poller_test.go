package poller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockTzkt struct {
	mock.Mock
}

func (m *mockTzkt) GetDelegationsByLevel(ctx context.Context, level uint64, dataChan chan<- *types.ChanMsg) error {
	args := m.Called(ctx, level, dataChan)
	if args.Get(0) == nil {
		dataChan <- &types.ChanMsg{
			Level: level,
			Data:  []types.FetchedDelegation{{Sender: types.Sender{Address: "tz1"}, Level: level, Amount: 1234}},
		}
	}
	return args.Error(0)
}

func (m *mockTzkt) SubscribeToHead(ctx context.Context, dataChan chan<- *types.ChanMsg, currentHead chan<- uint64, errorChan chan<- error) {
	m.Called(ctx, dataChan, currentHead, errorChan)
}

type mockStore struct {
	mock.Mock
}

func (m *mockStore) GetCurrentLevel(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

type mockConfig struct {
}

func (m *mockConfig) GetStartLevel() uint64 {
	return 0
}
func (m *mockConfig) GetRetryAttempts() int {
	return 2
}

func TestPoller_Run(t *testing.T) {

	t.Run("Nomical case", func(t *testing.T) {
		mockTzktInstance := new(mockTzkt)
		mockStoreInstance := new(mockStore)
		dataChan := make(chan *types.ChanMsg)
		errorChan := make(chan error)

		mockStoreInstance.On("GetCurrentLevel", mock.Anything).Return(uint64(100), nil)
		mockTzktInstance.On("SubscribeToHead", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			currentHead := args.Get(2).(chan<- uint64)
			currentHead <- 101
		})
		mockTzktInstance.On("GetDelegationsByLevel", mock.Anything, uint64(101), mock.Anything).Return(nil)

		cfg := &mockConfig{}

		poller := NewPoller(mockTzktInstance, dataChan, mockStoreInstance, cfg, errorChan)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		go poller.Run(ctx)

		select {
		case err := <-errorChan:
			assert.Fail(t, "Unexpected error", err)
		case msg := <-dataChan:
			assert.Equal(t, uint64(101), msg.Level)
			assert.False(t, msg.Reorg)
			assert.Len(t, msg.Data, 1)
			assert.Equal(t, "tz1", msg.Data[0].Sender.Address)
			assert.Equal(t, uint64(1234), msg.Data[0].Amount)
		case <-ctx.Done():
		}

		mockTzktInstance.AssertExpectations(t)
		mockStoreInstance.AssertExpectations(t)

	})
	t.Run("Subscribe to head error", func(t *testing.T) {

		mockTzktInstance := new(mockTzkt)
		mockStoreInstance := new(mockStore)
		dataChan := make(chan *types.ChanMsg)
		errorChan := make(chan error)

		mockTzktInstance.On("SubscribeToHead", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			errorChan := args.Get(3).(chan<- error)
			errorChan <- errors.New("couldn't connect to tzkt ws: connection failed")
		})

		cfg := &mockConfig{}

		poller := NewPoller(mockTzktInstance, dataChan, mockStoreInstance, cfg, errorChan)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		go poller.Run(ctx)

		assert.Equal(t, (<-errorChan).Error(), "maximum reconnection attempts reached: couldn't connect to tzkt ws: connection failed")

		mockTzktInstance.AssertExpectations(t)

	})
	t.Run("Get current level error", func(t *testing.T) {

		mockTzktInstance := new(mockTzkt)
		mockStoreInstance := new(mockStore)
		dataChan := make(chan *types.ChanMsg)
		errorChan := make(chan error)

		mockStoreInstance.On("GetCurrentLevel", mock.Anything).Return(uint64(100), errors.New("db error"))
		mockTzktInstance.On("SubscribeToHead", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			currentHead := args.Get(2).(chan<- uint64)
			currentHead <- 101
		})
		mockTzktInstance.On("GetDelegationsByLevel", mock.Anything, uint64(101), mock.Anything).Return(nil)

		cfg := &mockConfig{}

		poller := NewPoller(mockTzktInstance, dataChan, mockStoreInstance, cfg, errorChan)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		go poller.Run(ctx)

		// no retries for db errors
		assert.Equal(t, (<-errorChan).Error(), "Error getting current database level: db error")

		mockStoreInstance.AssertExpectations(t)
		mockTzktInstance.AssertExpectations(t)

	})

}

func TestPoller_getPastDelegations(t *testing.T) {
	mockTzktInstance := new(mockTzkt)
	mockStoreInstance := new(mockStore)
	dataChan := make(chan *types.ChanMsg, 3)
	errorChan := make(chan error)

	mockTzktInstance.On("GetDelegationsByLevel", mock.Anything, uint64(101), mock.Anything).Return(nil)
	mockTzktInstance.On("GetDelegationsByLevel", mock.Anything, uint64(102), mock.Anything).Return(nil)
	mockTzktInstance.On("GetDelegationsByLevel", mock.Anything, uint64(103), mock.Anything).Return(errors.New("some error"))

	cfg := &mockConfig{}

	poller := NewPoller(mockTzktInstance, dataChan, mockStoreInstance, cfg, errorChan)

	ctx := context.Background()
	err := poller.getPastDelegations(ctx, 101, 103)

	assert.EqualError(t, err, "Error fetching delegations for level 103: some error")
	assert.Equal(t, (<-dataChan).Level, uint64(101))
	assert.Equal(t, (<-dataChan).Level, uint64(102))

	mockTzktInstance.AssertExpectations(t)
}
