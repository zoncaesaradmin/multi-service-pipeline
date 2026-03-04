Feature: Test multiple rule modify cases
  Send rule config changes and relevant data message to check functionality.

  Scenario: IT_031_base_rule
    And set_all_needed_kafka_topics
    And send_input_config "rule_trans.json"
    And send_input_data "data_trans.json"
    And wait_till_data_received_with_timeout_sec 20
    And verify_if_record_has_same_fabric_as_input_data
    And verify_if_record_has_acknowledged
    And verify_if_record_has_custom_message
    And verify_if_record_has_severity
    And send_input_data "data_nomatch.json"
    And wait_till_data_received_with_timeout_sec 20
    And verify_if_record_has_same_fabric_as_input_data
    And verify_if_record_has_expected_acknowledged_value "false"
    And verify_if_record_has_expected_custom_message_value ""
    And verify_if_record_has_expected_severity_value "minor"

#  # test rule modifications
#  Scenario Outline: IT_032_table_rule_modifications
#    And set_all_needed_kafka_topics
#    And send_input_config "<rule>"
#    And send_input_data "<rec>"
#    And wait_till_data_received_with_timeout_sec 20
#    And verify_if_record_has_same_fabric_as_input_data
#    And verify_if_record_has_acknowledged
#    And verify_if_record_has_custom_message
#    And verify_if_record_has_severity
#
#    Examples:
#      | rule | rec |
#      | rule_2_conn.json | data_2_conn.json |
