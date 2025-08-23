package steps

import (
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
	consHandler := types.NewConsumerHandler(b.SuiteCtx.L)
	if err := consHandler.Start(); err != nil {
		b.SuiteCtx.L.Errorf("Failed to start consumer on topic %s", topic, err)
		return err
	}
	b.SuiteCtx.ConsHandler = consHandler
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
