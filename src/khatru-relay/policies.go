package main

import (
	"context"
	"time"

	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
)

func getRelayPolicies() []func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return []func(ctx context.Context, event *nostr.Event) (reject bool, msg string){
		policies.PreventLargeTags(120),
		policies.PreventTimestampsInTheFuture(time.Minute * 30),
		policies.EventIPRateLimiter(2, time.Minute*3, 10),
		// Add more policies here
	}
}
