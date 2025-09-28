Feature: Scale tests with same fabricName, category etc with max matches but different interface

  # test scale of rules changing category with one data record
  Scenario: IT_098_scale_rules_category
    And set_all_needed_kafka_topics
    And replicate_and_send_input_config_with_interface "scalerule3_maxmatch.json" 10
    And send_input_data "scaledata3_matchlast.json"
    And wait_till_data_received_with_timeout_sec 5
    And verify_if_record_has_acknowledged
    And verify_if_record_has_custom_message
    And verify_if_record_has_severity

# test scale of M rules with N data record with different category
 Scenario: IT_099_scale_rules_data_category
    And set_all_needed_kafka_topics
    And replicate_and_send_input_config_with_interface "scalerule3_maxmatch.json" 10
    And wait_for_seconds 5
    And replicate_and_send_input_data_with_interface "scaledata3_matchlast.json" 10
    And wait_till_data_received_with_timeout_sec 10
