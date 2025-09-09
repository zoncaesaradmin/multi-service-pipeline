package impl

import (
	"context"
	"fmt"
	"os"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"sync"
	"telemetry/utils/alert"
	"testgomodule/types"

	"google.golang.org/protobuf/proto"
)

// this is a custom suite context structure that can be passed to the steps
type CustomContext struct {
	L               logging.Logger
	InConfigTopic   string
	InDataTopic     string
	OutDataTopic    string
	ConsHandler     *ConsumerHandler
	ProducerHandler *ProducerHandler
	SentDataSize    int
	SentDataCount   int
	CurrentScenario string            // Track current scenario for trace ID
	ExampleData     map[string]string // Track current example data for scenario outlines
	SentDataMeta    types.SentDataMeta
	SentConfigMeta  types.SentConfigMeta
}

type ConsumerHandler struct {
	consumer      messagebus.Consumer
	logger        logging.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	receivedAll   bool
	receivedCount int
	expectedCount int
	expectedMap   map[string]string
	receivedMsg   messagebus.Message
	mutex         sync.Mutex
}

// NewInputHandler creates a new input handler
func NewConsumerHandler(logger logging.Logger) *ConsumerHandler {

	confFilename := utils.ResolveConfFilePath("kafka-consumer.yaml")
	kafkaConf := utils.LoadConfigMap(confFilename)
	consumer := messagebus.NewConsumer(kafkaConf, "prealertConsGroup"+os.Getenv("HOSTNAME"))

	return &ConsumerHandler{
		consumer: consumer,
		logger:   logger,
	}
}

// Start starts the input handler
func (i *ConsumerHandler) Start() error {
	topics := []string{"cisco_nir-prealerts"}
	//i.logger.Infow("Starting consumer handler", "topics", topics)

	// Create context for cancellation
	i.ctx, i.cancel = context.WithCancel(context.Background())

	// Register OnMessage callback
	i.consumer.OnMessage(func(message *messagebus.Message) {
		if message != nil {
			// Extract trace ID from message headers for trace-aware logging
			traceID := utils.ExtractTraceID(message.Headers)
			var msgLogger logging.Logger
			if traceID != "" {
				msgLogger = utils.WithTraceLoggerFromID(i.logger, traceID)
			} else {
				msgLogger = i.logger
			}

			expHeaders := i.GetExpectedHeaders()
			if EnsureMapMatches(expHeaders, message.Headers) {
				msgLogger.Infow("Received valid test data message", "headers", message.Headers, "expHeaders", expHeaders)
				i.receivedCount++
				if i.receivedCount == i.expectedCount {
					i.receivedAll = true
					i.receivedMsg = *message
				}
			} else {
				msgLogger.Debugw("Received some other data message", "headers", message.Headers, "expHeaders", expHeaders)
			}
			if err := i.consumer.Commit(context.Background(), message); err != nil {
				msgLogger.Warnw("Failed to commit message", "error", err)
			}
		}
	})

	// Subscribe to topics
	if err := i.consumer.Subscribe(topics); err != nil {
		i.logger.Errorf("failed to subscribe to topics: %w", err)
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	return nil
}

// Stop stops the input handler
func (i *ConsumerHandler) Stop() error {
	i.logger.Info("Stopping consumer handler")

	if i.cancel != nil {
		i.cancel()
	}

	if i.consumer != nil {
		if err := i.consumer.Close(); err != nil {
			i.logger.Errorw("Error closing consumer", "error", err)
			return err
		}
	}

	return nil
}

func (i *ConsumerHandler) GetReceivedAll() bool {
	return i.receivedAll
}

func (i *ConsumerHandler) SetReceivedAll(receivedAll bool) {
	i.receivedAll = receivedAll
}

func (i *ConsumerHandler) SetExpectedCount(count int) {
	i.expectedCount = count
}

func (i *ConsumerHandler) SetExpectedHeaders(expectedMap map[string]string) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.expectedMap = expectedMap
}

func (i *ConsumerHandler) GetExpectedHeaders() map[string]string {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.expectedMap
}

func (i *ConsumerHandler) GetReceivedMsgSize() int {
	return len(i.receivedMsg.Value)
}

func (i *ConsumerHandler) Reset() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.receivedCount = 0
	i.expectedCount = 0
	i.receivedAll = false
	i.receivedMsg = messagebus.Message{}
	i.expectedMap = make(map[string]string)
}

func (i *ConsumerHandler) VerifyDataField(field string, value interface{}) bool {
	aStream := &alert.AlertStream{}
	err := proto.Unmarshal(i.receivedMsg.Value, aStream)
	if err != nil {
		i.logger.Errorf("Failed to unmarshal input record: %w", err)
		return false
	}
	//i.logger.Infof("Data record: %+v", aStream)

	successCount := 0
	for _, aObj := range aStream.AlertObject {
		if i.checkFieldMatch(field, aObj, value) {
			successCount++
		}
	}
	return successCount == len(aStream.AlertObject)
}

// checkFieldMatch checks if a specific field in the alert object matches the expected value
func (i *ConsumerHandler) checkFieldMatch(field string, aObj interface{}, value interface{}) bool {
	switch field {
	case "acknowledged":
		return i.checkAcknowledged(aObj, value)
	case "ruleCustomRecoStr":
		return i.checkRuleCustomReco(aObj, value)
	case "severity":
		return i.checkSeverity(aObj, value)
	case "fabricName":
		return i.checkFabricName(aObj, value)
	default:
		i.logger.Warnw("Unknown field for verification", "field", field)
		return false
	}
}

// checkAcknowledged verifies the acknowledged field using reflection
func (i *ConsumerHandler) checkAcknowledged(aObj interface{}, value interface{}) bool {
	// Use reflection or interface{} to access fields dynamically
	if objVal, ok := aObj.(interface{ GetAcknowledged() bool }); ok {
		return objVal.GetAcknowledged() == value.(bool)
	}
	return false
}

// checkRuleCustomReco verifies the ruleCustomRecoStr field
func (i *ConsumerHandler) checkRuleCustomReco(aObj interface{}, value interface{}) bool {
	if objVal, ok := aObj.(interface{ GetRuleCustomRecoStr() []string }); ok {
		expectedValue := value.(string)
		for _, reco := range objVal.GetRuleCustomRecoStr() {
			//i.logger.Infof("Verifying ruleCustomRecoStr with actual=%s expected=%s", reco, expectedValue)
			if reco == expectedValue {
				return true
			}
		}
	}
	return false
}

// checkSeverity verifies the severity field
func (i *ConsumerHandler) checkSeverity(aObj interface{}, value interface{}) bool {
	if objVal, ok := aObj.(interface{ GetSeverity() string }); ok {
		return objVal.GetSeverity() == value.(string)
	}
	return false
}

// checkFabricName verifies the fabricName field
func (i *ConsumerHandler) checkFabricName(aObj interface{}, value interface{}) bool {
	if objVal, ok := aObj.(interface{ GetFabricName() string }); ok {
		return objVal.GetFabricName() == value.(string)
	}
	return false
}

type ProducerHandler struct {
	producer messagebus.Producer
	logger   logging.Logger
}

func NewProducerHandler(logger logging.Logger) *ProducerHandler {
	confFilename := utils.ResolveConfFilePath("kafka-producer.yaml")
	kafkaConf := utils.LoadConfigMap(confFilename)

	producer := messagebus.NewProducer(kafkaConf, "testAnomalyProducer"+os.Getenv("HOSTNAME"))

	return &ProducerHandler{
		producer: producer,
		logger:   logger,
	}
}
func (o *ProducerHandler) Start() error {
	//o.logger.Debug("Starting producer handler")
	return nil
}

func (o *ProducerHandler) Stop() error {
	o.logger.Debug("Stopping producer handler")

	if o.producer != nil {
		if err := o.producer.Close(); err != nil {
			o.logger.Errorw("Error closing producer", "error", err)
			return err
		}
	}
	return nil
}

func (o *ProducerHandler) Send(topic string, data []byte, headers map[string]string) error {
	message := &messagebus.Message{
		Topic:   topic,
		Value:   data,
		Headers: headers,
	}
	_, _, err := o.producer.Send(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", topic, err)
	}

	//o.logger.Debugw("Message sent successfully", "topic", topic, "size", len(data))

	return nil
}

func EnsureMapMatches(expected, data map[string]string) bool {
	for k, v := range expected {
		dv, ok := data[k]
		if !ok || dv != v {
			return false
		}
	}
	return true
}
