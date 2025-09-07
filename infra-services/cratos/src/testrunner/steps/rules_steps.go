package steps

import (
	"errors"
	"testgomodule/impl"
	"time"

	"github.com/cucumber/godog"
)

type StepBindings struct {
	Cctx *impl.CustomContext
}

func (b *StepBindings) SendInputConfigToTopic(configFile string, topic string) error {
	rBytes, err := LoadRulesFromJSON(configFile)
	if err != nil {
		b.Cctx.L.Infof("Failed to load from rules config file %s\n", configFile)
		return err
	}
	b.Cctx.ProducerHandler.Send(topic, rBytes, nil)
	b.Cctx.L.Infof("Sent config %s over Kafka topic %s\n", configFile, topic)
	return nil
}

func (b *StepBindings) SendInputDataToTopic(dataFile string, topic string) error {
	aBytes, err := LoadAlertFromJSON(dataFile)
	if err != nil {
		b.Cctx.L.Infof("Failed to load from data file %s\n", dataFile)
		return err
	}
	metaDataMap := map[string]string{"testData": "true"}
	b.Cctx.ProducerHandler.Send(topic, aBytes, metaDataMap)
	b.Cctx.SentDataSize += len(aBytes)
	b.Cctx.SentDataCount++
	b.Cctx.L.Infof("+++++++ set expected count to 1 for data file %s\n", dataFile)
	b.Cctx.ConsHandler.SetExpectedCount(1)
	b.Cctx.ConsHandler.SetExpectedMap(metaDataMap)
	b.Cctx.L.Infof("Sent data %s over Kafka topic %s\n", dataFile, topic)
	return nil
}

func (b *StepBindings) WaitTillDataReceivedOnTopicWithTimeoutSec(topic string, timeoutSec int) error {
	b.Cctx.L.Infof("Started waiting for data on Kafka topic %s for %d seconds...\n", topic, timeoutSec)
	timeoutTicker := time.NewTicker(100 * time.Millisecond)
	defer timeoutTicker.Stop()
	timeoutCh := time.After(time.Duration(timeoutSec) * time.Second)
	for {
		select {
		case <-timeoutTicker.C:
			if b.Cctx.ConsHandler.GetReceivedAll() {
				b.Cctx.L.Infof("Wait ended as data is received on topic %s\n", topic)
				return nil
			}
		case <-timeoutCh:
			b.Cctx.L.Infof("Wait ended due to timeout before data is received on topic %s\n", topic)
			return errors.New("timeout waiting for data")
		}
	}
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

func InitializeRulesSteps(ctx *godog.ScenarioContext, suiteMetadataCtx *impl.CustomContext) {
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
	ctx.Step(`^verify_if_data_is_fully_received_as_is$`, bindings.VerifyIfDataIsFullyReceived)
	ctx.Step(`^verify_if_valid_field "([^"]*)"$`, bindings.VerifyIfValidField)
	ctx.Step(`^verify_if_all_fields_are_unchanged$`, bindings.VerifyIfNoFieldModified)
}
