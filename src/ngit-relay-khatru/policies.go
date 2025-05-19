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

func getRelayPolicies(relay *khatru.Relay, domain string) []func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return []func(ctx context.Context, event *nostr.Event) (reject bool, msg string){
		policies.PreventLargeTags(120),
		policies.PreventTimestampsInTheFuture(time.Minute * 30),
		policies.EventIPRateLimiter(2, time.Minute*3, 10),
		RelatesToExistingRepoOrAllowedNewRepo(relay, domain),
	}
}

func RelatesToExistingRepoOrAllowedNewRepo(relay *khatru.Relay, domain string) func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
		// allow repository root events
		if event.Kind == nostr.KindRepositoryState || event.Kind == nostr.KindRepositoryAnnouncement {
			return false, ""
		}
		return RelatesToExistingEvent(relay, domain)(ctx, event)
	}
}

func RelatesToExistingEvent(relay *khatru.Relay, domain string) func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
		// Only accept announcement events when the ngit-relay instance is listed correctly
		if event.Kind == nostr.KindRepositoryAnnouncement {
			listed_in_clones := false
			for _, tag := range event.Tags {
				if len(tag) > 1 && tag[0] == "clones" {
					for _, val := range tag[1:] {
						if strings.Contains(val, "://"+domain) {
							listed_in_clones = true
							break
						}
					}
				}
			}
			listed_in_relays := false
			for _, tag := range event.Tags {
				if len(tag) > 1 && tag[0] == "relays" {
					for _, val := range tag[1:] {
						if strings.Contains(val, "://"+domain) {
							listed_in_relays = true
							break
						}
					}
				}
			}
			if listed_in_clones && listed_in_relays {
				return false, ""
			}
			return true, "repository announcement doesn't list ngit-relay in tags: clones and relays"
		}

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

		// create filters
		filters := make([]nostr.Filter, 0)

		// filter for referenced events
		filters = append(filters, nostr.Filter{
			IDs:   eventIds,
			Limit: 1, // We only need to know if at least one exists
		})
		// filter for referenced address poitners
		if len(eventPointers) > 0 {
			for _, eventpointer := range eventPointers {
				parts := strings.Split(eventpointer, ":")
				if len(parts) == 3 {
					kind, _ := strconv.Atoi(parts[0])
					var kinds []int
					// allow state events that match an announcement event
					if kind == nostr.KindRepositoryState {
						kinds = []int{nostr.KindRepositoryState, nostr.KindRepositoryAnnouncement}
					} else {
						kinds = []int{kind}
					}
					filters = append(filters, nostr.Filter{
						Kinds:   kinds,
						Authors: []string{parts[1]},
						Tags: nostr.TagMap{
							"d": []string{parts[2]},
						}, Limit: 1, // We only need to know if at least one exists
					})
				}
				if len(parts) == 2 {
					kind, _ := strconv.Atoi(parts[0])
					filters = append(filters, nostr.Filter{
						Kinds:   []int{kind},
						Authors: []string{parts[1]},
						Limit:   1, // We only need to know if at least one exists
					})
				}
			}
		}
		// filter for references to events
		filters = append(filters, nostr.Filter{
			Tags: nostr.TagMap{
				"e": eventIds,
				"E": eventIds,
			},
			Limit: 1, // We only need to know if at least one exists
		})
		// filter for references to address pointers
		if len(eventPointers) > 0 {
			filters = append(filters, nostr.Filter{
				Tags: nostr.TagMap{
					"a": eventPointers,
					"A": eventPointers,
				},
				Limit: 1, // We only need to know if at least one exists
			})
		}
		filters = append(filters, nostr.Filter{
			Tags: nostr.TagMap{
				"q": append(eventPointers, eventIds...),
			},
			Limit: 1, // We only need to know if at least one exists
		})

		// query CountEvents functions provided by the relay's storage backend
		for _, filter := range filters {
			for _, countFn := range relay.CountEvents {
				if countFn == nil {
					continue
				}
				count, err := countFn(ctx, filter)
				if err == nil && count > 0 {
					return false, "" // Found a reference, or referenced event
				}
			}
		}
		return true, "event does not relate to a stored repository"
	}
}
