package types

import (
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
)

// this is a custom suite context structure that can be passed to the steps
type CustomContext struct {
	L        logging.Logger
	Producer messagebus.Producer
	Consumer messagebus.Consumer
}
