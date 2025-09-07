package steps

import (
	"encoding/json"
	"io/ioutil"
	"telemetry/utils/alert"

	"google.golang.org/protobuf/proto"
)

// Reads JSON data from a file and converts it to an Alert protobuf message
func LoadAlertFromJSON(filename string) ([]byte, error) {
	inputDataFile := testFilePath(filename)
	data, err := ioutil.ReadFile(inputDataFile)
	if err != nil {
		return nil, err
	}
	var AlertStream alert.AlertStream
	if err := json.Unmarshal(data, &AlertStream); err != nil {
		return nil, err
	}
	aBytes, err := proto.Marshal(&AlertStream)
	if err != nil {
		return nil, err
	}
	return aBytes, nil
}

// Reads JSON rule config from a file and converts it to an Alert protobuf message
func LoadRulesFromJSON(filename string) ([]byte, error) {
	ruleBytes, err := ioutil.ReadFile(testFilePath(filename))
	if err != nil {
		return nil, err
	}
	return ruleBytes, nil
}

func testFilePath(file string) string {
	return "testdata/" + file
}
