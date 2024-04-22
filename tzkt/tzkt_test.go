package tzkt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/dipdup-net/go-lib/tzkt/data"
	"github.com/dipdup-net/go-lib/tzkt/events"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHttpClient struct {
	mock.Mock
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// MockWebSocketClient implements the WebSocketClient interface
type mockWebSocketClient struct {
	mock.Mock
}

func (m *mockWebSocketClient) Connect(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockWebSocketClient) SubscribeToHead() error {
	return m.Called().Error(0)
}

func (m *mockWebSocketClient) Listen() <-chan events.Message {
	args := m.Called()
	return args.Get(0).(chan events.Message)
}

func (m *mockWebSocketClient) Close() error {
	return m.Called().Error(0)
}

func TestGetDelegationsByLevel(t *testing.T) {
	client := new(mockHttpClient)
	tzkt := &Tzkt{
		url:           "https://fake.api.tzkt.io",
		client:        client,
		retryAttempts: 3,
	}

	delegations := []types.FetchedDelegation{{Level: 1234}}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(delegations)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(buf),
	}
	client.On("Do", mock.Anything).Return(resp, nil)

	dataChan := make(chan *types.ChanMsg, 1)
	err := tzkt.GetDelegationsByLevel(context.Background(), 1234, dataChan)
	assert.NoError(t, err)
	assert.Len(t, dataChan, 1)
	msg := <-dataChan
	assert.Equal(t, uint64(1234), msg.Level)
	assert.False(t, msg.Reorg)
	assert.Equal(t, delegations, msg.Data)
}

func TestSubscribeToHead(t *testing.T) {

	// Init channels
	dataChan := make(chan *types.ChanMsg, 10)
	currentHead := make(chan uint64, 10)
	errorChan := make(chan error, 10)
	messageChan := make(chan events.Message, 10)

	ctx := context.Background()

	t.Run("Nominal case", func(t *testing.T) {

		mockWsClient := new(mockWebSocketClient)
		client := new(mockHttpClient)

		tzkt := &Tzkt{
			url:           "https://fake.api.tzkt.io",
			wsClient:      mockWsClient,
			client:        client,
			retryAttempts: 3,
		}
		delegations := []types.FetchedDelegation{{Level: 101}}
		buf := new(bytes.Buffer)
		json.NewEncoder(buf).Encode(delegations)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(buf),
		}

		// Prepare client mocks
		client.On("Do", mock.Anything).Return(resp, nil)

		// Prepare wsClient mocks
		mockWsClient.On("Connect", ctx).Return(nil)
		mockWsClient.On("SubscribeToHead").Return(nil)
		mockWsClient.On("Listen").Return(messageChan)
		mockWsClient.On("Close").Return(nil)

		go tzkt.SubscribeToHead(ctx, dataChan, currentHead, errorChan)

		// Simulating messages
		go func() {
			messageChan <- events.Message{Type: events.MessageTypeState, State: 100}
			messageChan <- events.Message{Type: events.MessageTypeData, Channel: events.ChannelHead, Body: data.Head{Level: 101}}
			messageChan <- events.Message{Type: events.MessageTypeReorg, State: 99}
			close(messageChan)
		}()

		// Check outputs
		assert.Equal(t, uint64(100), <-currentHead)
		assert.Equal(t, uint64(101), (<-dataChan).Level)
		assert.Equal(t, true, (<-dataChan).Reorg)
		assert.Empty(t, errorChan)

	})

	t.Run("Connection failed", func(t *testing.T) {
		mockWsClient := new(mockWebSocketClient)
		client := new(mockHttpClient)

		tzkt := &Tzkt{
			url:           "https://fake.api.tzkt.io",
			wsClient:      mockWsClient,
			client:        client,
			retryAttempts: 3,
		}
		mockWsClient.On("Connect", mock.Anything).Return(errors.New("connection failed"))
		go tzkt.SubscribeToHead(ctx, dataChan, currentHead, errorChan)
		assert.Equal(t, "couldn't connect to tzkt ws: connection failed", (<-errorChan).Error())
	})

	t.Run("Subscription failed", func(t *testing.T) {
		mockWsClient := new(mockWebSocketClient)
		client := new(mockHttpClient)

		tzkt := &Tzkt{
			url:           "https://fake.api.tzkt.io",
			wsClient:      mockWsClient,
			client:        client,
			retryAttempts: 3,
		}
		mockWsClient.On("Connect", ctx).Return(nil)
		mockWsClient.On("Close").Return(nil)
		mockWsClient.On("SubscribeToHead").Return(errors.New("subscription failed"))

		go tzkt.SubscribeToHead(ctx, dataChan, currentHead, errorChan)
		assert.Equal(t, "couldn't subscribe to tzkt head: subscription failed", (<-errorChan).Error())
	})

}

func TestExecuteRequest(t *testing.T) {
	tests := []struct {
		name            string
		prepare         func(*mockHttpClient)
		expectedError   string
		expectedRetries int
	}{
		{
			name: "Network Error, should retry and fail",
			prepare: func(client *mockHttpClient) {
				client.On("Do", mock.Anything).Return(&http.Response{}, errors.New("network error")).Times(3)
			},
			expectedError:   "network error",
			expectedRetries: 3,
		},
		{
			name: "HTTP 500 Error, should retry and fail",
			prepare: func(client *mockHttpClient) {
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				client.On("Do", mock.Anything).Return(resp, nil).Times(3)
			},
			expectedError:   "non-200 status code: 500",
			expectedRetries: 3,
		},
		{
			name: "HTTP 200 OK, should succeed",
			prepare: func(client *mockHttpClient) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}
				client.On("Do", mock.Anything).Return(resp, nil).Once()
			},
			expectedError:   "",
			expectedRetries: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := new(mockHttpClient)
			tc.prepare(client)
			tzkt := &Tzkt{
				client:        client,
				url:           "https://fake.api.tzkt.io",
				retryAttempts: 3,
			}
			req, _ := http.NewRequest("GET", tzkt.url, nil)
			ctx := context.Background()

			resp, err := tzkt.executeRequest(ctx, req)
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
			client.AssertNumberOfCalls(t, "Do", tc.expectedRetries)
		})
	}
}
