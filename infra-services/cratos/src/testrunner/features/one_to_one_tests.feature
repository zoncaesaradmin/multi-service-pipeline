Feature: Test multiple cases of matching rule with data.
  Send rule config and relevant data message to check functionality.

  # common steps for each of the scenarios in this feature
  Background:
    And set_input_config_topic "cisco_nir-alertRules"
    And set_input_data_topic "cisco_nir-anomalies"
    And set_output_data_topic "cisco_nir-prealerts"
    And ensure_test_config_kafka_producer_is_ready
    And ensure_test_data_consumer_on_output_is_ready

  # test rule and matching data if it works fine
  Scenario Outline: IT_011_table
    And send_input_config "<rule>"
    And send_input_data "<rec>"
    And wait_till_data_received_with_timeout_sec 20
    And verify_if_valid_fabric
    And verify_if_record_has_acknowledged
    And verify_if_record_has_custom_message
    And verify_if_record_has_severity

    Examples:
      | rule | rec |
      | rule_2_conn.json | data_2_conn.json |
