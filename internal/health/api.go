package health

import "github.com/alexliesenfeld/health"

type Check = health.Check

type Checks interface {
	// AddLivenessCheck adds a check that will run when the service is being
	// probed for liveness. These checks are exposed via the health server on
	// the /healthz endpoint.
	AddLivenessCheck(check Check)

	// AddReadinessCheck adds a check that will run when the service is being
	// probed for readiness. These checks are exposed via the health server on
	// the /readyz endpoint.
	AddReadinessCheck(check Check)
}
