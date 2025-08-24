package steps

import (
	"testgomodule/types"

	"github.com/cucumber/godog"
)

type CommonStepBindings struct {
	SuiteCtx *types.CustomContext
}

func (b *CommonStepBindings) KafkaProducerReady() error {
	prodHandler := types.NewProducerHandler(b.SuiteCtx.L)
	if err := prodHandler.Start(); err != nil {
		b.SuiteCtx.L.Errorf("Failed to start producer: %w", err)
		return err
	}
	b.SuiteCtx.ProducerHandler = prodHandler
	b.SuiteCtx.L.Infof("Kafka producer started successfully.")
	// reset
	b.SuiteCtx.SentDataSize = 0
	b.SuiteCtx.SentDataCount = 0
	return nil
}

func (b *CommonStepBindings) KafkaConsumersStarted(topic string) error {
	consHandler := types.NewConsumerHandler(b.SuiteCtx.L)
	if err := consHandler.Start(); err != nil {
		b.SuiteCtx.L.Errorf("Failed to start consumer on topic %s", topic, err)
		return err
	}
	b.SuiteCtx.ConsHandler = consHandler
	b.SuiteCtx.L.Infof("Kafka consumers started on topic %s.", topic)
	// reset
	b.SuiteCtx.ConsHandler.Reset()
	return nil
}

func InitializeCommonSteps(ctx *godog.ScenarioContext, suiteCtx *types.CustomContext) {
	bindings := &CommonStepBindings{SuiteCtx: suiteCtx}
	ctx.Step(`^kafka producer publishing test config and test data is ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^the test kafka consumer is listening on kafka topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
	ctx.Step(`^ensure_test_config_kafka_producer_is_ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^ensure_test_data_kafka_consumer_on_topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
}
