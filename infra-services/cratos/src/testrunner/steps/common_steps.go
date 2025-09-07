package steps

import (
	"testgomodule/impl"
	"time"

	"github.com/cucumber/godog"
)

type CommonStepBindings struct {
	SuiteCtx *impl.CustomContext
}

func (b *CommonStepBindings) KafkaProducerReady() error {
	prodHandler := impl.NewProducerHandler(b.SuiteCtx.L)
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

func (b *CommonStepBindings) KafkaConsumerReady() error {
	return b.KafkaConsumersStarted(b.SuiteCtx.OutDataTopic)
}

func (b *CommonStepBindings) KafkaConsumersStarted(topic string) error {
	consHandler := impl.NewConsumerHandler(b.SuiteCtx.L)
	if err := consHandler.Start(); err != nil {
		b.SuiteCtx.L.Errorf("Failed to start consumer on topic %s", topic, err)
		return err
	}
	b.SuiteCtx.ConsHandler = consHandler
	time.Sleep(5 * time.Second) // wait for consumer to be ready
	b.SuiteCtx.L.Infof("Kafka consumers started on topic %s.", topic)
	// reset
	b.SuiteCtx.ConsHandler.Reset()
	return nil
}

func InitializeCommonSteps(ctx *godog.ScenarioContext, suiteCtx *impl.CustomContext) {
	bindings := &CommonStepBindings{SuiteCtx: suiteCtx}
	ctx.Step(`^kafka producer publishing test config and test data is ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^the test kafka consumer is listening on kafka topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
	ctx.Step(`^ensure_test_config_kafka_producer_is_ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^ensure_test_data_kafka_consumer_on_topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
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
	ctx.Step(`^ensure_test_data_consumer_on_output_is_ready$`, bindings.KafkaConsumerReady)
}
