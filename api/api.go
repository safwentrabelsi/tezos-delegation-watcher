package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/metrics"
	"github.com/safwentrabelsi/tezos-delegation-watcher/types"
	"github.com/sirupsen/logrus"
)

type APIServer struct {
	cfg   *config.ServerConfig
	store storeInterface
}

var log = logrus.WithField("module", "server")

// Created a specific interface for the server since we only need GetDelegations
// It makes it easier to mock
type storeInterface interface {
	GetDelegations(ctx context.Context, year string) ([]types.Delegation, error)
}

// NewAPIServer creates a new api server instance with the specified config and data store.
func NewAPIServer(cfg *config.ServerConfig, store storeInterface) *APIServer {
	return &APIServer{
		cfg:   cfg,
		store: store,
	}
}

// Run starts Server with the metrics server.
func (s *APIServer) Run() {
	router := gin.Default()

	// Setup middlewares
	router.Use(ValidateYearParam(s.cfg.GetMinValidYear()))

	metricRouter := gin.New()
	m := ginmetrics.GetMonitor()
	m.UseWithoutExposingEndpoint(router)
	m.SetMetricPath("/metrics")
	m.Expose(metricRouter)

	go func() {
		if err := metrics.Init(); err != nil {
			log.Errorf(fmt.Sprintf("Metrics init failed: %s", err))
		}
		log.Infof("Metrics server started at url http://%s:%d/metrics", s.cfg.GetHost(), s.cfg.GetMetricsPort())
		if err := metricRouter.Run(fmt.Sprintf(":%d", s.cfg.GetMetricsPort())); err != nil {
			log.Errorf("Metrics server stopped: %v", err)
		}

	}()

	router.GET("/xtz/delegations", s.handleGetDelegation)
	router.GET("/liveness", s.handleLiveness)
	if err := router.Run(s.cfg.GetListenAddress()); err != nil {
		log.Fatalf("API server stopped: %v", err)
	}
}

func (s *APIServer) handleGetDelegation(c *gin.Context) {

	year := c.Query("year")
	delegations, err := s.store.GetDelegations(c.Request.Context(), year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": delegations})
}

// handleLiveness responds with HTTP 200 OK to indicate that the service is live.
func (s *APIServer) handleLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
