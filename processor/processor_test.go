package processor

import (
	"context"
	"errors"
	"testing"

	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) SaveDelegations(ctx context.Context, delegations []types.FetchedDelegation) error {
	args := m.Called(ctx, delegations)
	return args.Error(0)
}

func (m *MockStore) DeleteDelegationsFromLevel(ctx context.Context, level uint64) error {
	args := m.Called(ctx, level)
	return args.Error(0)
}

func (m *MockStore) GetCurrentLevel(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockStore) GetDelegations(ctx context.Context, year string) ([]types.Delegation, error) {
	args := m.Called(ctx, year)
	return args.Get(0).([]types.Delegation), args.Error(1)
}

func TestProcessor_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockStore := new(MockStore)
	dataChan := make(chan *types.ChanMsg, 1)
	errorChan := make(chan error, 1)
	doneChan := make(chan bool, 1)
	processor := NewProcessor(mockStore, dataChan, errorChan)

	mockStore.On("SaveDelegations", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		doneChan <- true // Signal that SaveDelegations has completed
	})
	mockStore.On("DeleteDelegationsFromLevel", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		doneChan <- true // Signal that SaveDelegations has completed
	})

	go processor.Run(ctx)

	dataChan <- &types.ChanMsg{
		Reorg: false,
		Level: 100,
		Data:  []types.FetchedDelegation{{Timestamp: "2024-04-21T16:23:27Z", Amount: 1000, Sender: types.Sender{Address: "tz1"}, Level: 100}},
	}

	<-doneChan // Wait for signal that processing has completed
	mockStore.AssertCalled(t, "SaveDelegations", mock.Anything, mock.Anything)

	dataChan <- &types.ChanMsg{
		Reorg: true,
		Level: 100,
	}

	<-doneChan // Wait for signal that processing has completed
	mockStore.AssertCalled(t, "DeleteDelegationsFromLevel", mock.Anything, uint64(100))

}

func TestProcessor_DBError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockStore := new(MockStore)
	dataChan := make(chan *types.ChanMsg, 1)
	errorChan := make(chan error, 1)

	processor := NewProcessor(mockStore, dataChan, errorChan)

	mockStore.On("SaveDelegations", mock.Anything, mock.Anything).Return(errors.New("DB error"))
	mockStore.On("DeleteDelegationsFromLevel", mock.Anything, uint64(100)).Return(errors.New("DB error"))

	go processor.Run(ctx)

	t.Run("Save error", func(t *testing.T) {
		dataChan <- &types.ChanMsg{
			Reorg: false,
			Level: 100,
			Data:  []types.FetchedDelegation{{Timestamp: "2024-04-21T16:23:27Z", Amount: 1000, Sender: types.Sender{Address: "tz1"}, Level: 100}},
		}
		assert.Equal(t, (<-errorChan).Error(), "failed to save delegations: DB error")

	})
	t.Run("Delete error", func(t *testing.T) {
		dataChan <- &types.ChanMsg{
			Reorg: true,
			Level: 100,
		}
		assert.Equal(t, (<-errorChan).Error(), "failed to delete delegations: DB error")

	})

}
