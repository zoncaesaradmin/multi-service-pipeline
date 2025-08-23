package types

import "sharedgomodule/logging"

// this is a custom suite context structure that can be passed to the steps
type CustomContext struct {
	L logging.Logger
	// ... other shared state ...
}
