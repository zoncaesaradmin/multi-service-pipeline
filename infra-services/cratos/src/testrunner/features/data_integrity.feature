Feature: using code like steps
  Background:
    Given ensure_test_config_kafka_producer_is_ready
    And ensure_test_data_kafka_consumer_on_topic "output-topic"

  Scenario: Check for data loss and field integrity
    Given send_input_config_to_topic "sampleconfig.json" "rules-topic"
    And send_input_data_to_topic "sampledata2.json", "input-topic"
    When wait_till_data_received_on_topic_with_timeout_sec "output-topic", 2
    Then verify_if_data_is_fully_received
    And verify_if_valid_fabricname "fabricName"
    And verify_if_all_fields_are_unchanged
