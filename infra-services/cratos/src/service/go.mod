module servicegomodule

go 1.23.0

toolchain go1.24.6

require (
	github.com/joho/godotenv v1.5.1
	gopkg.in/yaml.v3 v3.0.1
	telemetry/utils/ruleenginelib v0.0.0-20240418123456-abcdef123456
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/grpc v1.74.2 // indirect
	google.golang.org/protobuf v1.36.7 // indirect
)

replace telemetry/utils/ruleenginelib => ../../../../client/golang/src/telemetry/utils/ruleenginelib
