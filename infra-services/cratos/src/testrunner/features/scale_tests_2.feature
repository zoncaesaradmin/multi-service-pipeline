Feature: Scale test

  # test scale of rules with one data record 
  Scenario: IT_091_scale_rules
    And set_input_config_topic "cisco_nir-alertRules"
    And set_input_data_topic "cisco_nir-anomalies"
    And set_output_data_topic "cisco_nir-prealerts"
    And replicate_and_send_input_config_with_category "scalerule2.json" "scale" 52
    And wait_for_seconds 5
    And send_input_data "scaledata2.json"
    And wait_till_data_received_with_timeout_sec 5
    And verify_if_record_has_acknowledged
    And verify_if_record_has_custom_message
    And verify_if_record_has_severity
