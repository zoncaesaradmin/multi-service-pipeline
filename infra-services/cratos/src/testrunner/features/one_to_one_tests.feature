Feature: Test multiple cases of matching rule with data.
  Send rule config and relevant data message to check functionality.

  # Kafka producer and consumer are now managed at feature level
  # No background steps needed - resources are shared across all scenarios

  # send all the rules first to save time for verification of impact on record data
  Scenario Outline: IT_010_table_rules
    And set_all_needed_kafka_topics
    And send_input_config "<rule>"

    Examples:
      | rule |
      | rule_2_conn.json |

  # send each record data and verify if it hits the corresponding rule above
  Scenario Outline: IT_010_table_reccords
    And set_all_needed_kafka_topics
    And send_input_data "<rec>"
    And wait_till_data_received_with_timeout_sec 20
    And verify_if_valid_fabric
    And verify_if_record_has_acknowledged
    And verify_if_record_has_custom_message
    And verify_if_record_has_severity

    Examples:
      | rec |
      | data_2_conn.json |
