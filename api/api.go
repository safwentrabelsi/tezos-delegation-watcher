package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
	"github.com/safwentrabelsi/tezos-delegation-watcher/config"
	"github.com/safwentrabelsi/tezos-delegation-watcher/store"
	"github.com/sirupsen/logrus"
)

type DelegationQueryParams struct {
	Year string `form:"year" binding:"omitempty,numeric"`
}

type APIServer struct {
	cfg   *config.ServerConfig
	store store.Storer
}

func NewAPIServer(cfg *config.ServerConfig, store store.Storer) *APIServer {
	return &APIServer{
		cfg:   cfg,
		store: store,
	}
}

func (s *APIServer) Run() {
	router := gin.Default()
	metricRouter := gin.New()
	m := ginmetrics.GetMonitor()
	m.UseWithoutExposingEndpoint(router)
	m.SetMetricPath("/metrics")
	m.Expose(metricRouter)

	go func() {
		logrus.Infof("Metrics server started at url http://localhost:%d/metrics", 8081)
		if err := metricRouter.Run(fmt.Sprintf(":%d", 8081)); err != nil {
			logrus.Errorf("Metrics server stopped: %v", err)
		}
	}()

	router.GET("/xtz/delegations", s.handleGetDelegation)
	if err := router.Run(s.cfg.GetListenAddress()); err != nil {
		logrus.Errorf("API server stopped: %v", err)
	}
}

func (s *APIServer) handleGetDelegation(c *gin.Context) {
	var params DelegationQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameter", "details": err.Error()})
		return
	}

	if err := validateYear(params.Year); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	delegations, err := s.store.GetDelegations(params.Year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": delegations})
}

// validateYear checks if the year is within a reasonable range
func validateYear(yearStr string) error {
	if yearStr == "" {
		return nil
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return err
	}

	currentYear := time.Now().Year()
	// Get 2018 from config
	// 2018 is the launch year of tezos mainnet
	if year < 2018 || year > currentYear {
		return fmt.Errorf("year must be between 2018 and %d", currentYear)
	}

	return nil
}
