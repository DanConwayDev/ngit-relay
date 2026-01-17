package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"go.uber.org/zap"

	"ngit-relay/shared"

	"github.com/nbd-wtf/go-nostr"
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

var commitID string

func main() {
	shared.Init("ngit-relay-khatru", true, true)
	logger := shared.L()

	// Disable go-nostr logging to stderr
	nostr.InfoLogger.SetOutput(io.Discard)
	nostr.DebugLogger.SetOutput(io.Discard)

	// Define flags for relay-data-dir and git-data-dir
	relay_data_path := flag.String("relay-data-dir", "", "Directory for relay data")
	git_data_path := flag.String("git-data-dir", "", "Directory for repositories data")
	blossom_data_path := flag.String("blossom-data-dir", "", "Directory for blossom data")

	// Parse the flags
	flag.Parse()

	// Check if the required arguments are provided
	if *relay_data_path == "" || *git_data_path == "" || *blossom_data_path == "" {
		flag.Usage()
		logger.Fatal("relay-data-dir, git-data-dir and blossom_data_path are required CLI arguments.")
	}

	config := Config{
		Domain:               getEnv("NGIT_DOMAIN"),
		RelayDataPath:        *relay_data_path,   // Dereference the pointer to get the string value
		GitDataPath:          *git_data_path,     // Dereference the pointer to get the string value
		BlossomDataPath:      *blossom_data_path, // Dereference the pointer to get the string value
		OwnerNpub:            getEnv("NGIT_OWNER_NPUB"),
		RelayName:            getEnv("NGIT_RELAY_NAME"),
		RelayDescription:     getEnv("NGIT_RELAY_DESCRIPTION"),
		BlossomMaxFileSizeMb: getEnvInt("NGIT_BLOSSOM_MAX_FILE_SIZE_MB", 100),
		BlossomMaxCapacityGb: getEnvInt("NGIT_BLOSSOM_MAX_CAPACITY_GB", 50),
	}
	OwnerPubkey, err := shared.GetPubkeyFromNpub(config.OwnerNpub)
	if err != nil {
		logger.Fatal("invalid NGIT_OWNER_NPUB", zap.Error(err))
	}

	// Create new relay
	relay := khatru.NewRelay()

	// Basic relay info (NIP-11)
	relay.Info.Name = config.RelayName
	relay.Info.PubKey = OwnerPubkey
	relay.Info.Description = config.RelayDescription
	relay.Info.Icon = ""
	relay.Info.SupportedNIPs = append(relay.Info.SupportedNIPs, 34)
	relay.Info.Software = "https://gitworkshop.dev/danconwaydev.com/ngit-relay"
	relay.Info.Version = "0.0.2"
	if commitID != "" {
		relay.Info.Version = relay.Info.Version + "-" + commitID
	}

	db := badger.BadgerBackend{Path: config.RelayDataPath}
	db.Init()
	relay.OnEventSaved = append(relay.OnEventSaved, EventReceiveHook(config.GitDataPath))
	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.ReplaceEvent = append(relay.ReplaceEvent, db.ReplaceEvent)
	relay.RejectEvent = append(relay.RejectEvent, getRelayPolicies(relay, config.Domain)...)
	relay.RejectConnection = append(relay.RejectConnection, ConnectionRateLimiterForOtherIPs(10, time.Minute, 2000))

	initBlossom(relay, config)

	// Start HTTP server on port 3334
	logger.Info("Starting nostr relay HTTP server", zap.String("address", ":3334"))
	if err := http.ListenAndServe(":3334", relay); err != nil {
		logger.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		shared.L().Fatal("Required environment variable not set", zap.String("key", key))
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	intValue, err := strconv.Atoi(valueStr)
	if err != nil {
		shared.L().Fatal("Invalid integer value for environment variable",
			zap.String("key", key),
			zap.String("value", valueStr),
			zap.Error(err),
		)
	}
	return intValue
}
