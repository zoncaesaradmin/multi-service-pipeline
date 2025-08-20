Feature: Rules roundtrip
  Background:
    Given the Kafka producers and consumers are started

  Scenario: Verify rules processing and output
    Given input config from "sampleconfig.json" and send over kafka topic "rules-topic"
    And data from "sampledata.json" and send over kafka topic "input-topic"
    When wait till the sent data is received on kafka topic "output-topic" until a timeout of 2 seconds
    Then verify if the data is fully received without loss
    And verify if data field "fabricName" is the same
    And verify if no field is modified as expected
