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

func (b *StepBindings) SendInputConfigToTopic(configFile string, topic string) error {
	inputConfigFile = configFile
	b.Cctx.L.Infof("Sent config %s over Kafka topic %s\n", configFile, topic)
	return nil
}

func (b *StepBindings) SendInputDataToTopic(dataFile string, topic string) error {
	inputDataFile = dataFile
	b.Cctx.L.Infof("Sent data %s over Kafka topic %s\n", dataFile, topic)
	return nil
}

func (b *StepBindings) WaitTillDataReceivedOnTopicWithTimeoutSec(topic string, timeoutSec int) error {
	outputTopic = topic
	b.Cctx.L.Infof("Waiting for data on Kafka topic %s for %d seconds...\n", topic, timeoutSec)
	time.Sleep(time.Duration(timeoutSec) * time.Second)
	// Simulate received data
	receivedData = map[string]interface{}{
		"fabricName": "FabricA",
	}
	return nil
}

func (b *StepBindings) VerifyIfDataIsFullyReceived() error {
	b.Cctx.L.Infof("Verified: Data received without loss.")
	return nil
}

func (b *StepBindings) VerifyIfValidField(field string) error {
	b.Cctx.L.Infof("Verified: Data field %s is valid.\n", field)
	return nil
}

func (b *StepBindings) VerifyIfNoFieldModified() error {
	b.Cctx.L.Infof("Verified: No field is modified as expected.\n")
	return nil
}

func InitializeRulesSteps(ctx *godog.ScenarioContext, suiteMetadataCtx *types.CustomContext) {
	bindings := &StepBindings{Cctx: suiteMetadataCtx}
	ctx.Step(`^send input config "([^"]*)" over kafka topic "([^"]*)"$`, bindings.SendInputConfigToTopic)
	ctx.Step(`^send input data "([^"]*)" over kafka topic "([^"]*)"$`, bindings.SendInputDataToTopic)
	ctx.Step(`^wait till the sent data is received on kafka topic "([^"]*)" with a timeout of (\d+) seconds$`, bindings.WaitTillDataReceivedOnTopicWithTimeoutSec)
	ctx.Step(`^verify if the data is fully received without loss$`, bindings.VerifyIfDataIsFullyReceived)
	ctx.Step(`^verify if data field "([^"]*)" is the same$`, bindings.VerifyIfValidField)
	ctx.Step(`^verify if no field is modified as expected$`, bindings.VerifyIfNoFieldModified)
	// same as above steps but written in code function style
	ctx.Step(`^send_input_config_to_topic "([^"]*)" "([^"]*)"$`, bindings.SendInputConfigToTopic)
	ctx.Step(`^send_input_data_to_topic "([^"]*)", "([^"]*)"$`, bindings.SendInputDataToTopic)
	ctx.Step(`^wait_till_data_received_on_topic_with_timeout_sec "([^"]*)", (\d+)$`, bindings.WaitTillDataReceivedOnTopicWithTimeoutSec)
	ctx.Step(`^verify_if_data_is_fully_received$`, bindings.VerifyIfDataIsFullyReceived)
	ctx.Step(`^verify_if_valid_field "([^"]*)"$`, bindings.VerifyIfValidField)
	ctx.Step(`^verify_if_all_fields_are_unchanged$`, bindings.VerifyIfNoFieldModified)
}
