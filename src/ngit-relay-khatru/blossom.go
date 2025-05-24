package main

import (
	"bytes"
	"context"
	"io"
	"ngit-relay/shared"
	"os"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/blossom"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

func initBlossom(relay *khatru.Relay, config Config) {

	logger := shared.L().With(zap.String("type", "Bossom"))

	bl := blossom.New(relay, "http://localhost:3334")
	bl_db := badger.BadgerBackend{Path: config.BlossomDataPath + "/db"}
	bl_db.Init()
	bl.Store = blossom.EventStoreBlobIndexWrapper{
		Store:      &bl_db,
		ServiceURL: bl.ServiceURL,
	}

	blob_path := config.BlossomDataPath + "/blobs/"

	fs := afero.NewOsFs()

	fs.MkdirAll(blob_path, os.ModeAppend)

	bl.StoreBlob = append(bl.StoreBlob, func(ctx context.Context, sha256 string, body []byte) error {
		logger.Debug("storing", zap.String("sha256", sha256))
		file, err := fs.Create(blob_path + "/" + sha256)
		if err != nil {
			logger.Error("error storing blob during fs.Create", zap.String("sha256", sha256), zap.Error((err)))
			return err
		}
		if _, err := io.Copy(file, bytes.NewReader(body)); err != nil {
			logger.Error("error storing blob with io.Copy", zap.String("sha256", sha256), zap.Error((err)))
			return err
		}
		return nil
	})
	bl.LoadBlob = append(bl.LoadBlob, func(ctx context.Context, sha256 string) (io.ReadSeeker, error) {
		logger.Warn("open", zap.String("sha256", sha256))
		return fs.Open(blob_path + "/" + sha256)
	})
	bl.DeleteBlob = append(bl.DeleteBlob, func(ctx context.Context, sha256 string) error {
		logger.Warn("delete", zap.String("sha256", sha256))
		return fs.Remove(blob_path + "/" + sha256)
	})

	total_stored, _ := getDirSize(fs, blob_path)

	bl.RejectUpload = append(bl.RejectUpload, func(ctx context.Context, event *nostr.Event, size int, ext string) (bool, string, int) {
		rejLogger := logger.With(zap.String("pubkey", event.PubKey), zap.String("ext", ext), zap.Int("size", size))

		// always allow uploads from owner
		if event.PubKey == nPubToPubkey(config.OwnerNpub) {
			total_stored = total_stored + size
			rejLogger.Debug("owner upload", zap.String("ext", ext), zap.Int("size", size))
			return false, ext, size
		}

		// check file size
		if config.BlossomMaxFileSizeMb > 0 && size > config.BlossomMaxFileSizeMb*1024*1024 {
			rejLogger.Warn("rejected blob - file too large", zap.String("ext", ext), zap.Int("size", size))
			return true, "file too large", 413
		}

		// capacity usage
		if config.BlossomMaxCapacityGb > 0 && (total_stored+size) > config.BlossomMaxCapacityGb*1024*1024*1024 {
			total_stored = total_stored + size
			rejLogger.Warn("rejected blob - blossom server full", zap.String("ext", ext), zap.Int("max_capacity_gb", config.BlossomMaxCapacityGb), zap.Int("total_stored", total_stored), zap.Int("size", size))
			return true, "blossom server full", 507
		}

		rejLogger.Debug("upload")
		return false, ext, size

	})

}

func getDirSize(fs afero.Fs, dirPath string) (int, error) {
	var totalSize int64
	err := afero.Walk(fs, dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return int(totalSize), nil
}

func nPubToPubkey(nPub string) string {
	_, v, err := nip19.Decode(nPub)
	if err != nil {
		panic(err)
	}
	return v.(string)
}
