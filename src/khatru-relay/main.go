package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
)

func main() {
	// Create new relay
	relay := khatru.NewRelay()

	// Basic relay info (NIP-11)
	relay.Info.Name = "ngit-relay"
	relay.Info.PubKey = ""
	relay.Info.Description = "Nostr relay powered by Khatru"
	relay.Info.Icon = ""

	db := badger.BadgerBackend{Path: "/khatru-data"}
	db.Init()
	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.ReplaceEvent = append(relay.ReplaceEvent, db.ReplaceEvent)
	relay.RejectEvent = append(relay.RejectEvent, getRelayPolicies()...)
	relay.RejectConnection = append(relay.RejectConnection, policies.ConnectionRateLimiter(1, time.Minute*5, 100))

	// Start HTTP server on port 3334
	fmt.Println("Running nostr relay on :3334")
	http.ListenAndServe(":3334", relay)
}
