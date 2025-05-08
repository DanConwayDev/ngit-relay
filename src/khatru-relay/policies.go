package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
)

func getRelayPolicies(relay *khatru.Relay) []func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return []func(ctx context.Context, event *nostr.Event) (reject bool, msg string){
		policies.PreventLargeTags(120),
		policies.PreventTimestampsInTheFuture(time.Minute * 30),
		policies.EventIPRateLimiter(2, time.Minute*3, 10),
		RelatesToExistingRepoOrAllowedNewRepo(relay),
	}
}

func RelatesToExistingRepoOrAllowedNewRepo(relay *khatru.Relay) func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
		// allow repository root events
		if event.Kind == nostr.KindRepositoryState || event.Kind == nostr.KindRepositoryAnnouncement {
			return false, ""
		}
		return RelatesToExistingEvent(relay)(ctx, event)
	}
}

func RelatesToExistingEvent(relay *khatru.Relay) func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
		// accept event that refers to a stored event, or is referenced by a stored event
		eventPointers := make([]string, 0)
		eventIds := make([]string, 0)

		// extract all events referenced
		for _, tag := range event.Tags {
			if len(tag) < 2 {
				continue
			}
			key := tag[0]
			value := tag[1]
			if key == "a" || key == "A" || (key == "q" && strings.Contains(value, ":")) {
				eventPointers = append(eventPointers, value)
			} else if key == "e" || key == "E" || (key == "q" && !strings.Contains(value, ":")) {
				eventIds = append(eventIds, value)
			}
		}

		// add current event
		if nostr.IsAddressableKind(event.Kind) {
			ptr := strconv.Itoa(event.Kind) + ":" + event.PubKey
			if dTag := event.Tags.Find("d"); len(dTag) > 1 {
				ptr += ":" + dTag[1]
			}
			eventPointers = append(eventPointers, ptr)
		} else {
			eventIds = append(eventIds, event.ID)
		}

		// create filter
		filter := nostr.Filter{
			Tags: nostr.TagMap{
				"e": eventIds,
				"E": eventIds,
				"a": eventPointers,
				"A": eventPointers,
				"q": append(eventPointers, eventIds...),
			},
			Limit: 1, // We only need to know if at least one exists
		}

		// query CountEvents functions provided by the relay's storage backend
		for _, countFn := range relay.CountEvents {
			if countFn == nil {
				continue
			}
			count, err := countFn(ctx, filter)
			if err == nil && count > 0 {
				return false, "" // Found a reference, or referenced event
			}
		}
		return true, "event does not relate to a stored repository"
	}
}
