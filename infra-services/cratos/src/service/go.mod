module servicegomodule

go 1.23.0

toolchain go1.24.6

require (
	github.com/joho/godotenv v1.5.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v3 v3.0.1
	telemetry/utils/alert v0.0.0-20240418123456-abcdef123456
	telemetry/utils/ruleenginelib v0.0.0-20240418123456-abcdef123456
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace telemetry/utils/ruleenginelib => ../../../../client/golang/src/telemetry/utils/ruleenginelib

replace telemetry/utils/alert => ../../../../client/golang/src/telemetry/utils/alert
