package steps

import (
	"testgomodule/types"

	"github.com/cucumber/godog"
)

type CommonStepBindings struct {
	SuiteCtx *types.CustomContext
}

func (b *CommonStepBindings) KafkaProducerReady() error {
	b.SuiteCtx.L.Info("Kafka producer is ready.")
	return nil
}

func (b *CommonStepBindings) KafkaProducersConsumersStarted() error {
	b.SuiteCtx.L.Info("Kafka producers and consumers started.")
	return nil
}

func InitializeCommonSteps(ctx *godog.ScenarioContext, suiteCtx *types.CustomContext) {
	bindings := &CommonStepBindings{SuiteCtx: suiteCtx}
	ctx.Step(`^kafka producer publishing test config and test data is ready$`, bindings.KafkaProducerReady)
	ctx.Step(`^the test kafka consumer is listening on kafka topic "([^"]*)"$`, bindings.KafkaProducersConsumersStarted)
}
