Feature: Data integrity
  Background:
    Given kafka producer publishing test config and test data is ready
    And the test kafka consumer is listening on kafka topic "cisco_nir-prealerts"

  Scenario: Check for data loss and field integrity
    Given send input config "rule_1.json" over kafka topic "cisco_nir-alertRules"
    And send input data "data_1_conn.json" over kafka topic "cisco_nir-anomalies"
    When wait till the sent data is received on kafka topic "cisco_nir-prealerts" with a timeout of 2 seconds
    Then verify if the data is fully received without loss
    And verify if data field "fabricName" is the same
    And verify if no field is modified as expected
