package main

import (
	"time"

	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
)

func getRelayPolicies() []khatru.RejectReasonOrError {
	return []khatru.RejectReasonOrError{
		policies.PreventLargeTags(120),
		policies.PreventTimestampsInTheFuture(time.Minute * 30),
		policies.EventIPRateLimiter(2, time.Minute*3, 10),
		// Add more policies here
	}
}
