package metrics

import (
	"fmt"

	"github.com/penglongli/gin-metrics/ginmetrics"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("module", "metrics")

const reorgMsgCountMetricsName = "watcher_received_reorg_messages_count"

// Init metrics
func Init() error {
	err := initReceivedReogMsgCount()
	if err != nil {
		return err
	}
	return nil
}

// I wanted to know how many reorg message are there
func initReceivedReogMsgCount() error {
	counter := &ginmetrics.Metric{
		Type:        ginmetrics.Counter,
		Name:        reorgMsgCountMetricsName,
		Description: "Received reorg messages count",
		Labels:      []string{},
	}
	err := ginmetrics.GetMonitor().AddMetric(counter)
	if err != nil {
		log.Error(fmt.Sprintf("Error adding metric: %s", err))
		return err
	}
	return nil
}

// ReorgMsgCountInc increments the counter reorg_message_count
func ReorgMsgCountInc() {
	err := ginmetrics.GetMonitor().GetMetric(reorgMsgCountMetricsName).Inc([]string{})
	if err != nil {
		log.Error(fmt.Sprintf("Error incrementing metric: %s", err))
	}
}
