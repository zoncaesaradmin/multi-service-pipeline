package steps

import (
	"encoding/json"
	"io/ioutil"
	"telemetry/utils/alert"
)

// Reads JSON data from a file and converts it to an Alert protobuf message
func LoadAlertFromJSON(filename string) (*alert.AlertStream, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var AlertStream alert.AlertStream
	if err := json.Unmarshal(data, &AlertStream); err != nil {
		return nil, err
	}
	return &AlertStream, nil
}
