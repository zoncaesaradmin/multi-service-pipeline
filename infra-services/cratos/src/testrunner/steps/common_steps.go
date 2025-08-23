package steps

import (
	"fmt"
	"os"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"testgomodule/types"

	"github.com/cucumber/godog"
)

type CommonStepBindings struct {
	SuiteCtx *types.CustomContext
}

func (b *CommonStepBindings) KafkaProducerReady() error {
	confFilename := utils.ResolveConfFilePath("kafka-producer.yaml")
	kafkaConf := utils.LoadConfigMap(confFilename)
	b.SuiteCtx.Producer = messagebus.NewProducer(kafkaConf)
	b.SuiteCtx.L.Info("Kafka producer is ready.")
	return nil
}

func (b *CommonStepBindings) KafkaConsumersStarted(topic string) error {
	confFilename := utils.ResolveConfFilePath("kafka-consumer.yaml")
	kafkaConf := utils.LoadConfigMap(confFilename)
	b.SuiteCtx.Consumer = messagebus.NewConsumer(kafkaConf, "cGroupPreAlerts"+os.Getenv("HOSTNAME"))
	// Subscribe to topics
	if err := b.SuiteCtx.Consumer.Subscribe([]string{"cisco_nir-prealerts"}); err != nil {
		b.SuiteCtx.L.Errorf("failed to subscribe to topics: %w", err)
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}
	b.SuiteCtx.L.Infof("Kafka consumers started on topic %s.", topic)
	return nil
}

func InitializeCommonSteps(ctx *godog.ScenarioContext, suiteCtx *types.CustomContext) {
	bindings := &CommonStepBindings{SuiteCtx: suiteCtx}
	ctx.Step(`^kafka producer publishing test config and test data is ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^the test kafka consumer is listening on kafka topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
	ctx.Step(`^ensure_test_config_kafka_producer_is_ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^ensure_test_data_kafka_consumer_on_topic "([^"]*)"$`, bindings.KafkaConsumersStarted)
}
