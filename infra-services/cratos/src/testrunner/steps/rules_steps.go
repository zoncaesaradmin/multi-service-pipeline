package steps

import (
	"encoding/json"
	"errors"
	"fmt"
	"sharedgomodule/utils"
	"strconv"
	"testgomodule/impl"
	"time"

	"github.com/cucumber/godog"
	"google.golang.org/protobuf/proto"
)

type StepBindings struct {
	Cctx *impl.CustomContext
}

func (b *StepBindings) SendInputConfig(configFile string) error {
	// Capture config file as example data for trace ID generation
	if b.Cctx.ExampleData == nil {
		b.Cctx.ExampleData = make(map[string]string)
	}
	b.Cctx.ExampleData["X"] = configFile

	return b.SendInputConfigToTopic(configFile, b.Cctx.InConfigTopic)
}

func (b *StepBindings) SendInputConfigToTopic(configFile string, topic string) error {
	// Create trace ID for config messages too (for consistency)
	traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
	traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)

	rBytes, configMeta, _, err := LoadRulesFromJSON(configFile)
	if err != nil {
		// Use trace-aware logging if scenario is available
		traceLogger.Infof("Failed to load from rules config file %s\n", configFile)
		return err
	}

	b.Cctx.SentConfigMeta = configMeta
	b.Cctx.ProducerHandler.Send(topic, rBytes, nil)
	traceLogger.Infof("Sent config %s over Kafka topic %s and trace ID in headers\n", configFile, topic)
	time.Sleep(5 * time.Second) // slight delay to ensure config is processed first
	return nil
}

func (b *StepBindings) SendInputData(dataFile string) error {
	// Capture data file as example data for trace ID generation
	if b.Cctx.ExampleData == nil {
		b.Cctx.ExampleData = make(map[string]string)
	}
	b.Cctx.ExampleData["X"] = dataFile

	return b.SendInputDataToTopic(dataFile, b.Cctx.InDataTopic)
}

func (b *StepBindings) SendInputDataToTopic(dataFile string, topic string) error {
	aBytes, dataMeta, _, err := LoadAlertFromJSON(dataFile)
	if err != nil {
		b.Cctx.L.Infof("Failed to load from data file %s\n", dataFile)
		return err
	}

	// Create trace ID from current scenario name and example data
	traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
	if traceID == "" {
		traceID = "testrunner-default-trace"
	}

	// Use trace-aware logging
	traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)

	metaDataMap := map[string]string{
		"testData":   "true",
		"X-Trace-Id": traceID,
	}

	b.Cctx.ConsHandler.SetExpectedHeaders(metaDataMap)
	b.Cctx.ConsHandler.SetExpectedCount(1)
	b.Cctx.SentDataSize += len(aBytes)
	b.Cctx.SentDataCount++
	b.Cctx.SentDataMeta = dataMeta
	b.Cctx.ProducerHandler.Send(topic, aBytes, metaDataMap)

	traceLogger.Infof("Sent data %s over Kafka topic %s and trace ID in headers\n", dataFile, topic)
	return nil
}

func (b *StepBindings) WaitForSeconds(seconds int) error {
	time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (b *StepBindings) SendScaleInputConfigWithFabric(configFile string, prefix string, count int) error {
	// Capture config file as example data for trace ID generation
	if b.Cctx.ExampleData == nil {
		b.Cctx.ExampleData = make(map[string]string)
	}
	topic := b.Cctx.InConfigTopic

	_, configMeta, rconfig, err := LoadRulesFromJSON(configFile)
	if err != nil {
		// Use trace-aware logging if scenario is available
		b.Cctx.L.Infof("Failed to load from rules config file %s\n", configFile)
		return err
	}

	for i := 1; i <= count; i++ {

		suffix := fmt.Sprintf("%v", i)
		b.Cctx.ExampleData["X"] = configFile + suffix
		fabricName := prefix + "-fabric-" + suffix
		ruleUUID := prefix + "-fabrule-uuid-" + suffix
		ruleName := prefix + "-fabrule-name-" + suffix

		//return b.SendInputConfigToTopic(configFile, b.Cctx.InConfigTopic)
		// Create trace ID for config messages too (for consistency)
		traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
		traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)

		for m, arule := range rconfig.AlertRules {
			rconfig.AlertRules[m].UUID = ruleUUID
			rconfig.AlertRules[m].Name = ruleName
			rconfig.AlertRules[m].Priority = arule.Priority + 1
			for n := range arule.AlertRuleMatchCriteria {
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].SiteId = fabricName
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].UUID = ruleUUID + "criteria" + fmt.Sprintf("%v", n)
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].AlertRuleId = ruleUUID
			}
		}
		rBytes, _ := json.Marshal(rconfig)

		b.Cctx.SentConfigMeta = configMeta
		b.Cctx.ProducerHandler.Send(topic, rBytes, nil)
		traceLogger.Infof("Sent config %s over Kafka topic %s with fabric %s\n", configFile, topic, fabricName)
	}
	time.Sleep(time.Duration(5+(count/10)) * time.Second) // slight delay to ensure config is processed first
	return nil

}

func (b *StepBindings) SendScaleInputDataWithFabric(dataFile string, prefix string, dcount int) error {
	topic := b.Cctx.InDataTopic
	expMetaDataMap := map[string]string{
		"testData": "true",
	}
	b.Cctx.ConsHandler.SetExpectedCount(dcount)
	b.Cctx.ConsHandler.SetExpectedHeaders(expMetaDataMap)

	// Create trace ID from current scenario name and example data
	traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
	if traceID == "" {
		traceID = "testrunner-default-trace"
	}

	// Use trace-aware logging
	traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)

	metaDataMap := map[string]string{
		"testData":   "true",
		"X-Trace-Id": traceID,
	}

	_, dataMeta, aStream, err := LoadAlertFromJSON(dataFile)
	if err != nil {
		b.Cctx.L.Infof("Failed to load from data file %s\n", dataFile)
		return err
	}

	for i := 1; i <= dcount; i++ {
		suffix := fmt.Sprintf("%v", i)
		b.Cctx.ExampleData["X"] = dataFile + suffix
		fabricName := prefix + "-fabric-" + suffix

		for _, aObj := range aStream.AlertObject {
			aObj.FabricName = fabricName
		}
		aBytes, _ := proto.Marshal(aStream)

		b.Cctx.SentDataSize += len(aBytes)
		b.Cctx.SentDataCount++
		b.Cctx.SentDataMeta = dataMeta
		b.Cctx.ProducerHandler.Send(topic, aBytes, metaDataMap)
		traceLogger.Infof("Sent data %s over Kafka topic %s for fabric %s\n", dataFile, topic, fabricName)
	}
	return nil

}

func (b *StepBindings) SendScaleInputConfigWithCategory(configFile string, prefix string, count int) error {
	// Capture config file as example data for trace ID generation
	if b.Cctx.ExampleData == nil {
		b.Cctx.ExampleData = make(map[string]string)
	}
	topic := b.Cctx.InConfigTopic

	_, configMeta, rconfig, err := LoadRulesFromJSON(configFile)
	if err != nil {
		// Use trace-aware logging if scenario is available
		b.Cctx.L.Infof("Failed to load from rules config file %s\n", configFile)
		return err
	}

	for i := 1; i <= count; i++ {

		suffix := fmt.Sprintf("%v", i)
		b.Cctx.ExampleData["X"] = configFile + suffix
		category := prefix + "-category-" + suffix
		ruleUUID := prefix + "-catrule-uuid-" + suffix
		ruleName := prefix + "-catrule-name-" + suffix

		//return b.SendInputConfigToTopic(configFile, b.Cctx.InConfigTopic)
		// Create trace ID for config messages too (for consistency)
		traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
		traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)

		for m, arule := range rconfig.AlertRules {
			rconfig.AlertRules[m].UUID = ruleUUID
			rconfig.AlertRules[m].Name = ruleName
			rconfig.AlertRules[m].Priority = arule.Priority + 1
			for n := range arule.AlertRuleMatchCriteria {
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].CategoryMatchCriteria =
					[]MatchCriteria{
						{ObjectType: "category", ValueEquals: category},
					}
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].UUID = ruleUUID + "criteria" + fmt.Sprintf("%v", n)
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].AlertRuleId = ruleUUID
			}
		}
		rBytes, _ := json.Marshal(rconfig)
		traceLogger.Infof("SREEK send rule - %s\n", string(rBytes))

		b.Cctx.SentConfigMeta = configMeta
		b.Cctx.ProducerHandler.Send(topic, rBytes, nil)
		traceLogger.Infof("Sent config %s over Kafka topic %s with category %s\n", configFile, topic, category)
	}
	time.Sleep(time.Duration(5+(count/10)) * time.Second) // slight delay to ensure config is processed first
	return nil
}

func (b *StepBindings) SendScaleInputDataWithCategory(dataFile string, prefix string, count int) error {
	topic := b.Cctx.InDataTopic
	expMetaDataMap := map[string]string{
		"testData": "true",
	}
	b.Cctx.ConsHandler.SetExpectedCount(count)
	b.Cctx.ConsHandler.SetExpectedHeaders(expMetaDataMap)

	// Create trace ID from current scenario name and example data
	traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
	if traceID == "" {
		traceID = "testrunner-default-trace"
	}
	// Use trace-aware logging
	traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)
	metaDataMap := map[string]string{
		"testData":   "true",
		"X-Trace-Id": traceID,
	}

	_, dataMeta, aStream, err := LoadAlertFromJSON(dataFile)
	if err != nil {
		b.Cctx.L.Infof("Failed to load from data file %s\n", dataFile)
		return err
	}

	for i := 1; i <= count; i++ {
		suffix := fmt.Sprintf("%v", i)
		b.Cctx.ExampleData["X"] = dataFile + suffix
		category := prefix + "-category-" + suffix

		for _, aObj := range aStream.AlertObject {
			aObj.Category = category
		}
		aBytes, _ := proto.Marshal(aStream)

		b.Cctx.SentDataSize += len(aBytes)
		b.Cctx.SentDataCount++
		b.Cctx.SentDataMeta = dataMeta
		b.Cctx.ProducerHandler.Send(topic, aBytes, metaDataMap)
		traceLogger.Infof("Sent data %s over Kafka topic %s for category %s\n", dataFile, topic, category)
	}
	return nil

}

func (b *StepBindings) SendScaleInputConfigWithInterface(configFile string, prefix string, count int) error {
	// Capture config file as example data for trace ID generation
	if b.Cctx.ExampleData == nil {
		b.Cctx.ExampleData = make(map[string]string)
	}
	topic := b.Cctx.InConfigTopic

	_, configMeta, rconfig, err := LoadRulesFromJSON(configFile)
	if err != nil {
		// Use trace-aware logging if scenario is available
		b.Cctx.L.Infof("Failed to load from rules config file %s\n", configFile)
		return err
	}

	for i := 1; i <= count; i++ {

		suffix := fmt.Sprintf("%v", i)
		b.Cctx.ExampleData["X"] = configFile + suffix
		category := prefix + "-category-" + suffix
		ruleUUID := prefix + "-catrule-uuid-" + suffix
		ruleName := prefix + "-catrule-name-" + suffix

		//return b.SendInputConfigToTopic(configFile, b.Cctx.InConfigTopic)
		// Create trace ID for config messages too (for consistency)
		traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
		traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)

		for m, arule := range rconfig.AlertRules {
			rconfig.AlertRules[m].UUID = ruleUUID
			rconfig.AlertRules[m].Name = ruleName
			rconfig.AlertRules[m].Priority = arule.Priority + 1
			for n := range arule.AlertRuleMatchCriteria {
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].CategoryMatchCriteria =
					[]MatchCriteria{
						{ObjectType: "category", ValueEquals: category},
					}
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].UUID = ruleUUID + "criteria" + fmt.Sprintf("%v", n)
				rconfig.AlertRules[m].AlertRuleMatchCriteria[n].AlertRuleId = ruleUUID
			}
		}
		rBytes, _ := json.Marshal(rconfig)
		traceLogger.Infof("SREEK send rule - %s\n", string(rBytes))

		b.Cctx.SentConfigMeta = configMeta
		b.Cctx.ProducerHandler.Send(topic, rBytes, nil)
		traceLogger.Infof("Sent config %s over Kafka topic %s with category %s\n", configFile, topic, category)
	}
	time.Sleep(time.Duration(5+(count/10)) * time.Second) // slight delay to ensure config is processed first
	return nil
}

func (b *StepBindings) SendScaleInputDataWithInterface(dataFile string, prefix string, count int) error {
	topic := b.Cctx.InDataTopic
	expMetaDataMap := map[string]string{
		"testData": "true",
	}
	b.Cctx.ConsHandler.SetExpectedCount(count)
	b.Cctx.ConsHandler.SetExpectedHeaders(expMetaDataMap)

	// Create trace ID from current scenario name and example data
	traceID := utils.CreateContextualTraceID(b.Cctx.CurrentScenario, b.Cctx.ExampleData)
	if traceID == "" {
		traceID = "testrunner-default-trace"
	}
	// Use trace-aware logging
	traceLogger := utils.WithTraceLoggerFromID(b.Cctx.L, traceID)
	metaDataMap := map[string]string{
		"testData":   "true",
		"X-Trace-Id": traceID,
	}

	_, dataMeta, aStream, err := LoadAlertFromJSON(dataFile)
	if err != nil {
		b.Cctx.L.Infof("Failed to load from data file %s\n", dataFile)
		return err
	}

	for i := 1; i <= count; i++ {
		suffix := fmt.Sprintf("%v", i)
		b.Cctx.ExampleData["X"] = dataFile + suffix
		category := prefix + "-category-" + suffix

		for _, aObj := range aStream.AlertObject {
			aObj.Category = category
		}
		aBytes, _ := proto.Marshal(aStream)

		b.Cctx.SentDataSize += len(aBytes)
		b.Cctx.SentDataCount++
		b.Cctx.SentDataMeta = dataMeta
		b.Cctx.ProducerHandler.Send(topic, aBytes, metaDataMap)
		traceLogger.Infof("Sent data %s over Kafka topic %s for category %s\n", dataFile, topic, category)
	}
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
	receivedSize := b.Cctx.ConsHandler.GetReceivedMsgSize()
	if receivedSize != b.Cctx.SentDataSize {
		return errors.New("data size mismatch: sent " + strconv.Itoa(b.Cctx.SentDataSize) + " bytes, received " + strconv.Itoa(receivedSize) + " bytes")
	}
	b.Cctx.L.Infof("Verified: Data received without loss.")
	return nil
}

func (b *StepBindings) VerifyIfValidFabric() error {
	// fetch sent fabric name from meta
	fabricName := b.Cctx.SentDataMeta.FabricName

	if !b.Cctx.ConsHandler.VerifyDataField("fabricName", fabricName) {
		return errors.New("fabric does not match")
	}
	b.Cctx.L.Infof("Verified: Record has fabric '%s'.\n", fabricName)
	return nil
}

func (b *StepBindings) VerifyIfAcknowledged() error {
	// fetch sent fabric name from meta
	ack := b.Cctx.SentConfigMeta.ActionAcknowledged

	if !b.Cctx.ConsHandler.VerifyDataField("acknowledged", ack) {
		return errors.New("field 'acknowledged' is not true as expected")
	}
	b.Cctx.L.Infof("Verified: Record has acknowledged '%v'\n", ack)
	return nil
}

func (b *StepBindings) VerifyIfRecordHasCustomMessage() error {
	// fetch sent custom message from meta
	expectedMessage := b.Cctx.SentConfigMeta.ActionCustomRecoValue

	if !b.Cctx.ConsHandler.VerifyDataField("ruleCustomRecoStr", expectedMessage) {
		return errors.New("custom message does not match")
	}
	b.Cctx.L.Infof("Verified: Record has custom message '%s'.\n", expectedMessage)
	return nil
}

func (b *StepBindings) VerifyIfRecordHasSeverity() error {
	// fetch sent severity from meta
	expectedSeverity := b.Cctx.SentConfigMeta.ActionSeverityValue

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
	// same as above steps but written in code function style
	ctx.Step(`^send_input_config_to_topic "([^"]*)" "([^"]*)"$`, bindings.SendInputConfigToTopic)
	ctx.Step(`^send_input_data_to_topic "([^"]*)", "([^"]*)"$`, bindings.SendInputDataToTopic)
	ctx.Step(`^wait_till_data_received_on_topic_with_timeout_sec "([^"]*)", (\d+)$`, bindings.WaitTillDataReceivedOnTopicWithTimeoutSec)
	ctx.Step(`^verify_if_data_is_fully_received_as_is$`, bindings.VerifyIfDataIsFullyReceived)

	ctx.Step(`^verify_if_valid_fabric$`, bindings.VerifyIfValidFabric)
	ctx.Step(`^verify_if_record_has_acknowledged$`, bindings.VerifyIfAcknowledged)
	ctx.Step(`^verify_if_record_has_custom_message$`, bindings.VerifyIfRecordHasCustomMessage)
	ctx.Step(`^verify_if_record_has_severity$`, bindings.VerifyIfRecordHasSeverity)

	ctx.Step(`^send_input_config "([^"]*)"$`, bindings.SendInputConfig)
	ctx.Step(`^send_input_data "([^"]*)"$`, bindings.SendInputData)
	ctx.Step(`^wait_till_data_received_with_timeout_sec (\d+)$`, bindings.WaitTillDataReceivedWithTimeoutSec)

	ctx.Step(`^wait_for_seconds (\d+)$`, bindings.WaitForSeconds)
	ctx.Step(`^replicate_and_send_input_config_with_fabricname "([^"]*)" "([^"]*)" (\d+)$`, bindings.SendScaleInputConfigWithFabric)
	ctx.Step(`^replicate_and_send_input_data_with_fabricname "([^"]*)" "([^"]*)" (\d+)$`, bindings.SendScaleInputDataWithFabric)

	ctx.Step(`^replicate_and_send_input_config_with_category "([^"]*)" "([^"]*)" (\d+)$`, bindings.SendScaleInputConfigWithCategory)
	ctx.Step(`^replicate_and_send_input_data_with_category "([^"]*)" "([^"]*)" (\d+)$`, bindings.SendScaleInputDataWithCategory)

	ctx.Step(`^replicate_and_send_input_config_with_interface "([^"]*)" "([^"]*)" (\d+)$`, bindings.SendScaleInputConfigWithInterface)
	ctx.Step(`^replicate_and_send_input_data_with_interface "([^"]*)" "([^"]*)" (\d+)$`, bindings.SendScaleInputDataWithInterface)
}
