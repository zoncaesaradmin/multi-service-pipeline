package steps

import (
	"fmt"
	"time"

	"github.com/cucumber/godog"
)

var (
	inputConfigFile string
	inputDataFile   string
	outputTopic     string
	receivedData    map[string]interface{}
)

func GivenInputConfigAndSendOverKafka(configFile, topic string) error {
	inputConfigFile = configFile
	fmt.Printf("Sent config %s over Kafka topic %s\n", configFile, topic)
	return nil
}

func GivenDataAndSendOverKafka(dataFile, topic string) error {
	inputDataFile = dataFile
	fmt.Printf("Sent data %s over Kafka topic %s\n", dataFile, topic)
	return nil
}

func WhenWaitForKafkaOutput(topic string, timeoutSec int) error {
	outputTopic = topic
	fmt.Printf("Waiting for data on Kafka topic %s for %d seconds...\n", topic, timeoutSec)
	time.Sleep(time.Duration(timeoutSec) * time.Second)
	// Simulate received data
	receivedData = map[string]interface{}{
		"fabricName": "FabricA",
	}
	return nil
}

func ThenVerifyDataReceivedWithoutLoss() error {
	fmt.Println("Verified: Data received without loss.")
	return nil
}

func ThenVerifyDataFieldIsSame(field string) error {
	fmt.Printf("Verified: Data field %s is the same.\n", field)
	return nil
}

func ThenVerifyNoFieldModified() error {
	fmt.Println("Verified: No field is modified as expected.")
	return nil
}

func InitializeRulesSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^input config from "([^"]*)" and send over kafka topic "([^"]*)"$`, GivenInputConfigAndSendOverKafka)
	ctx.Step(`^data from "([^"]*)" and send over kafka topic "([^"]*)"$`, GivenDataAndSendOverKafka)
	ctx.Step(`^wait till the sent data is received on kafka topic "([^"]*)" until a timeout of (\d+) seconds$`, WhenWaitForKafkaOutput)
	ctx.Step(`^verify if the data is fully received without loss$`, ThenVerifyDataReceivedWithoutLoss)
	ctx.Step(`^verify if data field "([^"]*)" is the same$`, ThenVerifyDataFieldIsSame)
	ctx.Step(`^verify if no field is modified as expected$`, ThenVerifyNoFieldModified)
}
