Feature: Data integrity
  Background:
    Given kafka producer publishing test config and test data is ready
    And the test kafka consumer is listening on kafka topic "output-topic"

  Scenario: Check for data loss and field integrity
    Given send input config "sampleconfig.json" over kafka topic "rules-topic"
    And send input data "sampledata2.json" over kafka topic "input-topic"
    When wait till the sent data is received on kafka topic "output-topic" with a timeout of 2 seconds
    Then verify if the data is fully received without loss
    And verify if data field "fabricName" is the same
    And verify if no field is modified as expected
