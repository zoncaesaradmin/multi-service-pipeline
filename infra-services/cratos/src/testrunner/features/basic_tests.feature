# This feature file uses BDD Gherkin syntax to write steps, but in code style
# with function like single keyword followed by parameters
# In BDD, the keywords "Given", "When", "And", "Then" are only for readability.
# replacing one with another does not impact execution.
# So, to avoid any thinking, we can just use one keyword always say "And"

Feature: Basic test for data reception without hitting any rule
  Tests base functionalities of rule engine service for various inputs.
  Test pod sends rule config and data records and verifies from output topic

  # common steps for each of the scenarios in this feature
  Background:
    And ensure_test_config_kafka_producer_is_ready
    And ensure_test_data_kafka_consumer_on_topic "cisco_nir-prealerts"

  # test if basic input and output of service are working fine
  Scenario: IT_001_Basic_data_reception_without_rules
    And send_input_config_to_topic "sampleconfig.json" "cisco_nir-alertRules"
    And send_input_data_to_topic "samplefulldata1.json", "cisco_nir-anomalies"
    And wait_till_data_received_on_topic_with_timeout_sec "cisco_nir-prealerts", 10
    And verify_if_data_is_fully_received_as_is
    And verify_if_valid_field "fabricName"
    And verify_if_all_fields_are_unchanged

  # test if basic input and output of service are working fine
  Scenario: IT_002_Basic_data_reception_with_simple_rule
    And send_input_config_to_topic "rule_title_conn.json" "cisco_nir-alertRules"
    And send_input_data_to_topic "data_conn_anomaly.json", "cisco_nir-anomalies"
    And wait_till_data_received_on_topic_with_timeout_sec "cisco_nir-prealerts", 10
    And verify_if_data_is_fully_received_as_is
    And verify_if_valid_field "fabricName"
    And verify_if_all_fields_are_unchanged