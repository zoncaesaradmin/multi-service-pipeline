package processing

import (
	"context"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	relib "telemetry/utils/ruleenginelib"
	"time"
)

type RuleEngineConfig struct {
	RulesTopic  string
	PollTimeout time.Duration
	Logging     logging.LoggerConfig
}

type RuleEngineHandler struct {
	consumer messagebus.Consumer
	config   RuleEngineConfig
	logger   logging.Logger
	reInst   *relib.RuleEngine
	inputCh  chan<- *models.ChannelMessage
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewRuleEngineHandler(config RuleEngineConfig, logger logging.Logger) *RuleEngineHandler {
	// use simple filename - path resolution is handled by messagebus config loader
	consumer := messagebus.NewConsumer("kafka-consumer.yaml", "ruleConsGroup"+utils.GetEnv("HOSTNAME", ""))
	lInfo := relib.LoggerInfo{
		ServiceName: consumer,
		Logger:      logger,
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RuleEngineHandler{
		config:  config,
		logger:  logger,
		inputCh: inputCh,
		ctx:     ctx,
		cancel:  cancel,
	}
}
