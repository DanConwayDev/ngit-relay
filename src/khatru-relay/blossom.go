package main

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/blossom"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/spf13/afero"
)

func initBlossom(relay *khatru.Relay, config Config) {

	bl := blossom.New(relay, "http://localhost:3334")
	bl_db := badger.BadgerBackend{Path: config.blossom_data_path + "/db"}
	bl_db.Init()
	bl.Store = blossom.EventStoreBlobIndexWrapper{
		Store:      &bl_db,
		ServiceURL: bl.ServiceURL,
	}

	blob_path := config.blossom_data_path + "/blobs/"

	fs := afero.NewOsFs()

	fs.MkdirAll(blob_path, os.ModeAppend)

	bl.StoreBlob = append(bl.StoreBlob, func(ctx context.Context, sha256 string, body []byte) error {
		file, err := fs.Create(blob_path + sha256)
		if err != nil {
			return err
		}
		if _, err := io.Copy(file, bytes.NewReader(body)); err != nil {
			return err
		}
		return nil
	})
	bl.LoadBlob = append(bl.LoadBlob, func(ctx context.Context, sha256 string) (io.ReadSeeker, error) {
		return fs.Open(blob_path + sha256)
	})
	bl.DeleteBlob = append(bl.DeleteBlob, func(ctx context.Context, sha256 string) error {
		return fs.Remove(blob_path + sha256)
	})

	total_stored, _ := getDirSize(fs, blob_path)

	bl.RejectUpload = append(bl.RejectUpload, func(ctx context.Context, event *nostr.Event, size int, ext string) (bool, string, int) {

		// always allow uploads from owner
		if event.PubKey == nPubToPubkey(config.owner_npub) {
			total_stored = total_stored + size
			return false, ext, size
		}

		// check file size
		if size > config.blossom_max_file_size_mb*1024*1024 {
			return true, "file too large", 413
		}

		// capacity usage
		if (total_stored + size) > config.blossom_max_capacity_gb*1024*1024*1024 {
			total_stored = total_stored + size
			return true, "blossom server full", 507
		}

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
