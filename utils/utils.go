package utils

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

func HandleErrors(ctx context.Context, cancel context.CancelFunc, errors chan error) {
	select {
	case err := <-errors:
		logrus.Errorf("Fatal error: %v", err)
		cancel()
		os.Exit(1)
	case <-ctx.Done():
		logrus.Info("Shutdown completed")
	}
}
