package api

import (
	"github.com/sdslabs/beastv4/api/middleware"
)

var (
	isPayloadLoggingEnabled = middleware.IsPayloadLoggingEnabled
	requestResponseLogger   = middleware.RequestResponseLogger
)
