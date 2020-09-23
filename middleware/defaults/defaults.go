package defaults

import (
	"github.com/onbyzerollc/pubsub"
	"github.com/onbyzerollc/pubsub/middleware/audit"
	"github.com/onbyzerollc/pubsub/middleware/logrus"
	"github.com/onbyzerollc/pubsub/middleware/opentracing"
	"github.com/onbyzerollc/pubsub/middleware/prometheus"
	"github.com/onbyzerollc/pubsub/middleware/recover"
)

// Middleware is a helper to import the default middleware for pubsub
var Middleware = []pubsub.Middleware{
	logrus.Middleware{},
	prometheus.Middleware{},
	opentracing.Middleware{},
	audit.Middleware{},
	recover.Middleware{},
}

// MiddlewareWithRecovery returns the default middleware but allows
// you to inject a function for dealing with panics
func MiddlewareWithRecovery(fn recover.RecoveryHandlerFunc) []pubsub.Middleware {
	return []pubsub.Middleware{
		logrus.Middleware{},
		prometheus.Middleware{},
		opentracing.Middleware{},
		audit.Middleware{},
		recover.Middleware{RecoveryHandlerFunc: fn},
	}
}
