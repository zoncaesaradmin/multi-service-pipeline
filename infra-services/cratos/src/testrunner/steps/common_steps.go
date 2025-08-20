package steps

import (
	"fmt"

	"github.com/cucumber/godog"
)

func KafkaProducersConsumersStarted() error {
	// Simulate starting Kafka producers/consumers
	fmt.Println("Kafka producers and consumers started.")
	return nil
}

func InitializeCommonSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the Kafka producers and consumers are started$`, KafkaProducersConsumersStarted)
}
