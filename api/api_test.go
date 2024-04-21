package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetDelegations(ctx context.Context, year string) ([]types.Delegation, error) {
	args := m.Called(ctx, year)
	return args.Get(0).([]types.Delegation), args.Error(1)
}

func TestHandleGetDelegation_NominalCase(t *testing.T) {
	router := gin.New()
	router.Use(gin.Recovery())
	mockStore := new(MockStore)
	server := NewAPIServer(&config.ServerConfig{}, mockStore)

	router.GET("/xtz/delegations", server.handleGetDelegation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/xtz/delegations?year=2024", nil)

	expectedDelegations := []types.Delegation{
		{Timestamp: "2024-04-21T16:23:27Z", Amount: 100, Delegator: "tz1", Block: 1},
	}
	mockStore.On("GetDelegations", mock.Anything, "2024").Return(expectedDelegations, nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":[{"timestamp":"2024-04-21T16:23:27Z","amount":100,"delegator":"tz1","block":1}]}`, w.Body.String())
	mockStore.AssertExpectations(t)

}

func TestHandleGetDelegation_DBError(t *testing.T) {
	router := gin.New()
	router.Use(gin.Recovery())

	mockStore := new(MockStore)
	server := NewAPIServer(&config.ServerConfig{}, mockStore)

	router.GET("/xtz/delegations", server.handleGetDelegation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/xtz/delegations?year=2024", nil)
	mockStore.On("GetDelegations", mock.Anything, "2024").Return([]types.Delegation{}, errors.New("database error"))

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"database error"}`, w.Body.String())
	mockStore.AssertExpectations(t)
}

func TestValidateYearParam(t *testing.T) {
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/test", ValidateYearParam(2018), func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	t.Run("Nomical case", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test?year=2019", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Test non numeric year", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test?year=abc", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.JSONEq(t, `{"error":"Year must be a valid number"}`, w.Body.String())
	})

	t.Run("Test out of range year", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test?year=1000", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.JSONEq(t, fmt.Sprintf(`{"error":"Year must be between 2018 and %d"}`, time.Now().Year()), w.Body.String())
	})

}
