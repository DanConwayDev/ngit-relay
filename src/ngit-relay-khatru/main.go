package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
)

type Config struct {
	Domain               string
	RelayDataPath        string
	GitDataPath          string
	BlossomDataPath      string
	OwnerNpub            string `json:"pubkey"`
	RelayName            string
	RelayDescription     string
	BlossomMaxFileSizeMb int
	BlossomMaxCapacityGb int
}

func main() {

	// Define flags for relay-data-dir and git-data-dir
	relay_data_path := flag.String("relay-data-dir", "", "Directory for relay data")
	git_data_path := flag.String("git-data-dir", "", "Directory for repositories data")
	blossom_data_path := flag.String("blossom-data-dir", "", "Directory for blossom data")

	// Parse the flags
	flag.Parse()

	// Check if the required arguments are provided
	if *relay_data_path == "" || *git_data_path == "" || *blossom_data_path == "" {
		fmt.Println("relay-data-dir, git-data-dir and blossom_data_path are required.")
		flag.Usage()
		return
	}

	config := Config{
		Domain:               getEnv("DOMAIN"),
		RelayDataPath:        *relay_data_path,   // Dereference the pointer to get the string value
		GitDataPath:          *git_data_path,     // Dereference the pointer to get the string value
		BlossomDataPath:      *blossom_data_path, // Dereference the pointer to get the string value
		OwnerNpub:            getEnv("OWNER_NPUB"),
		RelayName:            getEnv("RELAY_NAME"),
		RelayDescription:     getEnv("RELAY_DESCRIPTION"),
		BlossomMaxFileSizeMb: getEnvInt("BLOSSOM_MAX_FILE_SIZE_MB", 100),
		BlossomMaxCapacityGb: getEnvInt("BLOSSOM_MAX_CAPACITY_GB", 50),
	}

	// Create new relay
	relay := khatru.NewRelay()

	// Basic relay info (NIP-11)
	relay.Info.Name = config.RelayName
	relay.Info.PubKey = config.OwnerNpub
	relay.Info.Description = config.RelayDescription
	relay.Info.Icon = ""

	db := badger.BadgerBackend{Path: config.RelayDataPath}
	db.Init()
	relay.OnEventSaved = append(relay.OnEventSaved, EventRecieveHook(relay, config.GitDataPath))
	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.ReplaceEvent = append(relay.ReplaceEvent, db.ReplaceEvent)
	relay.RejectEvent = append(relay.RejectEvent, getRelayPolicies(relay, config.Domain)...)
	relay.RejectConnection = append(relay.RejectConnection, policies.ConnectionRateLimiter(1, time.Minute*5, 100))

	initBlossom(relay, config)

	// Start HTTP server on port 3334
	fmt.Println("Running nostr relay on :3334")
	http.ListenAndServe(":3334", relay)
}

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatalf("Environment variable %s not set", key)
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}
		return intValue
	}
	return defaultValue
}
