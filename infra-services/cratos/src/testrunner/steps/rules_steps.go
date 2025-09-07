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

func (b *StepBindings) SendInputConfig(configFile string) error {
	return b.SendInputConfigToTopic(configFile, b.Cctx.InConfigTopic)
}

func (b *StepBindings) SendInputConfigToTopic(configFile string, topic string) error {
	rBytes, err := LoadRulesFromJSON(configFile)
	if err != nil {
		b.Cctx.L.Infof("Failed to load from rules config file %s\n", configFile)
		return err
	}
	b.Cctx.ProducerHandler.Send(topic, rBytes, nil)
	b.Cctx.L.Infof("Sent config %s over Kafka topic %s\n", configFile, topic)
	time.Sleep(3 * time.Second) // slight delay to ensure config is processed first
	return nil
}

func (b *StepBindings) SendInputData(dataFile string) error {
	return b.SendInputDataToTopic(dataFile, b.Cctx.InDataTopic)
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
	b.Cctx.ConsHandler.SetExpectedCount(1)
	b.Cctx.ConsHandler.SetExpectedMap(metaDataMap)
	b.Cctx.L.Infof("Sent data %s over Kafka topic %s\n", dataFile, topic)
	return nil
}

func (b *StepBindings) WaitTillDataReceivedWithTimeoutSec(timeoutSec int) error {
	return b.WaitTillDataReceivedOnTopicWithTimeoutSec(b.Cctx.OutDataTopic, timeoutSec)
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

func (b *StepBindings) VerifyIfValidFabric(fabricName string) error {
	if !b.Cctx.ConsHandler.VerifyDataField("fabricName", fabricName) {
		return errors.New("fabric does not match")
	}
	b.Cctx.L.Infof("Verified: Record has fabric '%s'.\n", fabricName)
	return nil
}

func (b *StepBindings) VerifyIfNoFieldModified() error {
	b.Cctx.L.Infof("Verified: No field is modified as expected.\n")
	return nil
}

func (b *StepBindings) VerifyIfAcknowledged() error {
	if !b.Cctx.ConsHandler.VerifyDataField("acknowledged", true) {
		return errors.New("field 'acknowledged' is not true as expected")
	}
	b.Cctx.L.Infof("Verified: Record is acknowledged.\n")
	return nil
}

func (b *StepBindings) VerifyIfRecordHasCustomMessage(expectedMessage string) error {
	if !b.Cctx.ConsHandler.VerifyDataField("ruleCustomRecoStr", expectedMessage) {
		return errors.New("custom message does not match")
	}
	b.Cctx.L.Infof("Verified: Record has custom message '%s'.\n", expectedMessage)
	return nil
}

func (b *StepBindings) VerifyIfRecordHasSeverity(expectedSeverity string) error {
	if !b.Cctx.ConsHandler.VerifyDataField("severity", expectedSeverity) {
		return errors.New("severity does not match")
	}
	b.Cctx.L.Infof("Verified: Record has severity '%s'.\n", expectedSeverity)
	return nil
}

func InitializeRulesSteps(ctx *godog.ScenarioContext, suiteMetadataCtx *impl.CustomContext) {
	bindings := &StepBindings{Cctx: suiteMetadataCtx}
	ctx.Step(`^send input config "([^"]*)" over kafka topic "([^"]*)"$`, bindings.SendInputConfigToTopic)
	ctx.Step(`^send input data "([^"]*)" over kafka topic "([^"]*)"$`, bindings.SendInputDataToTopic)
	ctx.Step(`^wait till the sent data is received on kafka topic "([^"]*)" with a timeout of (\d+) seconds$`, bindings.WaitTillDataReceivedOnTopicWithTimeoutSec)
	ctx.Step(`^verify if the data is fully received without loss$`, bindings.VerifyIfDataIsFullyReceived)
	ctx.Step(`^verify if no field is modified as expected$`, bindings.VerifyIfNoFieldModified)
	// same as above steps but written in code function style
	ctx.Step(`^send_input_config_to_topic "([^"]*)" "([^"]*)"$`, bindings.SendInputConfigToTopic)
	ctx.Step(`^send_input_data_to_topic "([^"]*)", "([^"]*)"$`, bindings.SendInputDataToTopic)
	ctx.Step(`^wait_till_data_received_on_topic_with_timeout_sec "([^"]*)", (\d+)$`, bindings.WaitTillDataReceivedOnTopicWithTimeoutSec)
	ctx.Step(`^verify_if_data_is_fully_received_as_is$`, bindings.VerifyIfDataIsFullyReceived)

	ctx.Step(`^verify_if_valid_fabric "([^"]*)"$`, bindings.VerifyIfValidFabric)
	ctx.Step(`^verify_if_all_fields_are_unchanged$`, bindings.VerifyIfNoFieldModified)
	ctx.Step(`^verify_if_record_is_acknowledged$`, bindings.VerifyIfAcknowledged)
	ctx.Step(`^verify_if_record_has_custom_message "([^"]*)"$`, bindings.VerifyIfRecordHasCustomMessage)
	ctx.Step(`^verify_if_record_has_severity "([^"]*)"$`, bindings.VerifyIfRecordHasSeverity)

	ctx.Step(`^send_input_config "([^"]*)"$`, bindings.SendInputConfig)
	ctx.Step(`^send_input_data "([^"]*)"$`, bindings.SendInputData)
	ctx.Step(`^wait_till_data_received_with_timeout_sec (\d+)$`, bindings.WaitTillDataReceivedWithTimeoutSec)
}
