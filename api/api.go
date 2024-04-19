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

var log = logrus.WithField("module", "server")

func NewAPIServer(cfg *config.ServerConfig, store store.Storer) *APIServer {
	return &APIServer{
		cfg:   cfg,
		store: store,
	}
}

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
		log.Infof("Metrics server started at url http://%s:%d/metrics", s.cfg.GetHost(), s.cfg.GetMetricsPort())
		if err := metricRouter.Run(fmt.Sprintf(":%d", s.cfg.GetMetricsPort())); err != nil {
			log.Errorf("Metrics server stopped: %v", err)
		}
	}()

	router.GET("/xtz/delegations", s.handleGetDelegation)
	if err := router.Run(s.cfg.GetListenAddress()); err != nil {
		log.Errorf("API server stopped: %v", err)
	}
}

func (s *APIServer) handleGetDelegation(c *gin.Context) {

	year := c.Query("year")
	delegations, err := s.store.GetDelegations(year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": delegations})
}

func ValidateYearParam(minValidYear int) gin.HandlerFunc {
	return func(c *gin.Context) {
		yearStr := c.Query("year")
		if yearStr != "" {
			year, err := strconv.Atoi(yearStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Year must be a valid number"})
				c.Abort()
				return
			}

			currentYear := time.Now().Year()
			if year < minValidYear || year > currentYear {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Year must be between %d and %d", minValidYear, currentYear)})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
