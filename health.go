package sprout

import (
	"github.com/levelfourab/sprout-go/internal/health"
)

type HealthCheck = health.Check

// AsLivenessCheck takes a function that returns a HealthCheck and annotates it
// to be invoked as a liveness check. Liveness checks will be made available
// on the health server at the path /healthz.
//
// Example:
//
//	fx.Provide(sprout.AsLivenessCheck(func(depsHere...) sprout.HealthCheck {
//		return sprout.HealthCheck{
//			Name: "example",
//			Check: func(ctx context.Context) error {
//				return nil
//			},
//		}
//	}))
func AsLivenessCheck(f any) any {
	return health.AsLivenessCheck(f)
}

// AsReadinessCheck takes a function that returns a HealthCheck and annotates it
// to be invoked as a readiness check. Readiness checks will be made available
// on the health server at the path /readyz.
//
// Example:
//
//	fx.Provide(sprout.asReadinessCheck(func(depsHere...) sprout.HealthCheck {
//		return sprout.HealthCheck{
//			Name: "example",
//			Check: func(ctx context.Context) error {
//				return nil
//			},
//		}
//	}))
func AsReadinessCheck(f any) any {
	return health.AsReadinessCheck(f)
}
