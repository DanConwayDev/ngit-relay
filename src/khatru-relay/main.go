package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
)

func main() {

	type Config struct {
		relay_data_path string
		git_data_path   string
	}
	// Define flags for relay-data-dir and git-data-dir
	relay_data_path := flag.String("relay-data-dir", "", "Directory for relay data")
	git_data_path := flag.String("git-data-dir", "", "Directory for repositories data")

	// Parse the flags
	flag.Parse()

	// Check if the required arguments are provided
	if *relay_data_path == "" || *git_data_path == "" {
		fmt.Println("Both relay-data-dir and git-data-dir are required.")
		flag.Usage()
		return
	}
	config := Config{
		relay_data_path: *relay_data_path, // Dereference the pointer to get the string value
		git_data_path:   *git_data_path,   // Dereference the pointer to get the string value
	}

	// Create new relay
	relay := khatru.NewRelay()

	// Basic relay info (NIP-11)
	relay.Info.Name = "ngit-relay"
	relay.Info.PubKey = ""
	relay.Info.Description = "Nostr relay powered by Khatru"
	relay.Info.Icon = ""

	db := badger.BadgerBackend{Path: config.relay_data_path}
	db.Init()
	relay.OnEventSaved = append(relay.OnEventSaved, EventRecieveHook(config.git_data_path))
	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.ReplaceEvent = append(relay.ReplaceEvent, db.ReplaceEvent)
	relay.RejectEvent = append(relay.RejectEvent, getRelayPolicies(relay)...)
	relay.RejectConnection = append(relay.RejectConnection, policies.ConnectionRateLimiter(1, time.Minute*5, 100))

	// Start HTTP server on port 3334
	fmt.Println("Running nostr relay on :3334")
	http.ListenAndServe(":3334", relay)
}
