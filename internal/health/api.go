package health

import "github.com/alexliesenfeld/health"

type Check = health.Check

type Checks interface {
	AddLivenessCheck(check Check)

	AddReadinessCheck(check Check)
}
