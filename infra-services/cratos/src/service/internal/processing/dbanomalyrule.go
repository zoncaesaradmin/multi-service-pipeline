package processing

import (
	"context"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
)

func InitDBProcessing(ctx context.Context, l logging.Logger, inputCh chan<- *models.ChannelMessage) {
}

func StopDBProcessing(l logging.Logger) {
}
