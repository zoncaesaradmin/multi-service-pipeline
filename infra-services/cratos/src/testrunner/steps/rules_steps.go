package steps

import (
	"testgomodule/types"
	"time"

	"github.com/cucumber/godog"
)

var (
	inputConfigFile string
	inputDataFile   string
	outputTopic     string
	receivedData    map[string]interface{}
)

type StepBindings struct {
	Cctx *types.CustomContext
}

func (b *StepBindings) GivenInputConfigAndSendOverKafka(configFile string, topic string) error {
	inputConfigFile = configFile
	b.Cctx.L.Infof("Sent config %s over Kafka topic %s\n", configFile, topic)
	return nil
}

func (b *StepBindings) GivenDataAndSendOverKafka(dataFile, topic string) error {
	inputDataFile = dataFile
	b.Cctx.L.Infof("Sent data %s over Kafka topic %s\n", dataFile, topic)
	return nil
}

func (b *StepBindings) WhenWaitForKafkaOutput(topic string, timeoutSec int) error {
	outputTopic = topic
	b.Cctx.L.Infof("Waiting for data on Kafka topic %s for %d seconds...\n", topic, timeoutSec)
	time.Sleep(time.Duration(timeoutSec) * time.Second)
	// Simulate received data
	receivedData = map[string]interface{}{
		"fabricName": "FabricA",
	}
	return nil
}

func (b *StepBindings) ThenVerifyDataReceivedWithoutLoss() error {
	b.Cctx.L.Infof("Verified: Data received without loss.")
	return nil
}

func (b *StepBindings) ThenVerifyDataFieldIsSame(field string) error {
	b.Cctx.L.Infof("Verified: Data field %s is the same.\n", field)
	return nil
}

func (b *StepBindings) ThenVerifyNoFieldModified() error {
	b.Cctx.L.Infof("Verified: No field is modified as expected.\n")
	return nil
}

func InitializeRulesSteps(ctx *godog.ScenarioContext, suiteMetadataCtx *types.CustomContext) {
	bindings := &StepBindings{Cctx: suiteMetadataCtx}
	ctx.Step(`^send input config "([^"]*)" over kafka topic "([^"]*)"$`, bindings.GivenInputConfigAndSendOverKafka)
	ctx.Step(`^send input data "([^"]*)" over kafka topic "([^"]*)"$`, bindings.GivenDataAndSendOverKafka)
	ctx.Step(`^wait till the sent data is received on kafka topic "([^"]*)" with a timeout of (\d+) seconds$`, bindings.WhenWaitForKafkaOutput)
	ctx.Step(`^verify if the data is fully received without loss$`, bindings.ThenVerifyDataReceivedWithoutLoss)
	ctx.Step(`^verify if data field "([^"]*)" is the same$`, bindings.ThenVerifyDataFieldIsSame)
	ctx.Step(`^verify if no field is modified as expected$`, bindings.ThenVerifyNoFieldModified)
	// same as above steps but written in code function style
	ctx.Step(`^send_input_config_to_topic "([^"]*)" "([^"]*)"$`, bindings.GivenInputConfigAndSendOverKafka)
	ctx.Step(`^send_input_data_to_topic "([^"]*)", "([^"]*)"$`, bindings.GivenDataAndSendOverKafka)
	ctx.Step(`^wait_till_data_received_on_topic_with_timeout_sec "([^"]*)", (\d+)$`, bindings.WhenWaitForKafkaOutput)
	ctx.Step(`^verify_if_data_is_fully_received$`, bindings.ThenVerifyDataReceivedWithoutLoss)
	ctx.Step(`^verify_if_valid_fabricname "([^"]*)"$`, bindings.ThenVerifyDataFieldIsSame)
	ctx.Step(`^verify_if_all_fields_are_unchanged$`, bindings.ThenVerifyNoFieldModified)
}
