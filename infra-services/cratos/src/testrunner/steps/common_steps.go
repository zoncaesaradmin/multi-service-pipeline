package steps

import (
	"sharedgomodule/utils"
	"testgomodule/impl"
	"time"

	"github.com/cucumber/godog"
)

type CommonStepBindings struct {
	SuiteCtx *impl.CustomContext
}

func (b *CommonStepBindings) KafkaProducerReady() error {
	// Create trace-aware logger with contextual trace ID
	traceID := utils.CreateContextualTraceID(b.SuiteCtx.CurrentScenario, b.SuiteCtx.ExampleData)
	traceLogger := utils.WithTraceLoggerFromID(b.SuiteCtx.L, traceID)

	prodHandler := impl.NewProducerHandler(traceLogger, b.SuiteCtx.ExecGroupIndex)
	if err := prodHandler.Start(); err != nil {
		traceLogger.Errorf("Failed to start producer: %w", err)
		return err
	}
	b.SuiteCtx.ProducerHandler = prodHandler
	traceLogger.Infof("Kafka producer started successfully.")
	// reset
	b.SuiteCtx.SentDataSize = 0
	b.SuiteCtx.SentDataCount = 0
	return nil
}

func (b *CommonStepBindings) KafkaConsumerReady() error {
	return b.KafkaConsumersStarted(b.SuiteCtx.OutDataTopic)
}

func (b *CommonStepBindings) KafkaConsumersStarted(topic string) error {
	// Create trace-aware logger with contextual trace ID
	traceID := utils.CreateContextualTraceID(b.SuiteCtx.CurrentScenario, b.SuiteCtx.ExampleData)
	traceLogger := utils.WithTraceLoggerFromID(b.SuiteCtx.L, traceID)

	consHandler := impl.NewConsumerHandler(traceLogger, b.SuiteCtx.ExecGroupIndex)
	if err := consHandler.Start(); err != nil {
		traceLogger.Errorf("Failed to start consumer on topic %s", topic, err)
		return err
	}
	b.SuiteCtx.ConsHandler = consHandler
	time.Sleep(5 * time.Second) // wait for consumer to be ready
	traceLogger.Infof("Kafka consumer started on topic %s.", topic)
	// reset
	b.SuiteCtx.ConsHandler.Reset()
	return nil
}

func InitializeCommonSteps(ctx *godog.ScenarioContext, suiteCtx *impl.CustomContext) {
	bindings := &CommonStepBindings{SuiteCtx: suiteCtx}
	// below two steps can be used if any specific scenario needs to start/stop producer/consumer
	ctx.Step(`^kafka producer publishing test config and test data is ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^the test kafka consumer is listening on kafka topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
	ctx.Step(`^ensure_test_config_kafka_producer_is_ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^ensure_test_data_kafka_consumer_on_topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
	ctx.Step(`^ensure_test_data_consumer_on_output_is_ready$`, bindings.KafkaConsumerReady)
	ctx.Step(`^set_input_config_topic "([^"]*)"$`, func(topic string) error {
		bindings.SuiteCtx.InConfigTopic = topic
		return nil
	})
	ctx.Step(`^set_input_data_topic "([^"]*)"$`, func(topic string) error {
		bindings.SuiteCtx.InDataTopic = topic
		return nil
	})
	ctx.Step(`^set_output_data_topic "([^"]*)"$`, func(topic string) error {
		bindings.SuiteCtx.OutDataTopic = topic
		return nil
	})
	ctx.Step(`^set_all_needed_kafka_topics`, func() error {
		bindings.SuiteCtx.InConfigTopic = utils.GetEnv("PROCESSING_RULES_TOPIC", "cisco_nir-alertRules")
		bindings.SuiteCtx.InDataTopic = utils.GetEnv("PROCESSING_INPUT_TOPIC", "cisco_nir-alertInput")
		bindings.SuiteCtx.OutDataTopic = utils.GetEnv("PROCESSING_OUTPUT_TOPIC", "cisco_nir-alertOutput")
		return nil
	})
}
