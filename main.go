// Package vaultdump uploads vaultPass encrypted data to a Telegram channel.
// The data could be used later for restoration or DiRT excersises.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

func uploadFile(file, channel string) error {
	// Fail early if the path is not set.
	if file == "" {
		return fmt.Errorf("no file is provided.")
	}

	// Fail if the channel is not set.
	if channel == "" {
		return fmt.Errorf("no upload recepient set.")
	}

	// The performUpload will be called after client initialization.
	performUpload := func(ctx context.Context, client *telegram.Client) error {
		// Raw MTProto API client, allows making raw RPC calls.
		api := tg.NewClient(client)

		// Helper for uploading. Automatically uses big file upload when needed.
		u := uploader.NewUploader(api)

		// Helper for sending messages.
		sender := message.NewSender(api).WithUploader(u)

		// Uploading directly from path. Note that you can do it from
		// io.Reader or buffer, see From* methods of uploader.
		fmt.Printf("Uploading vaultPass dump %q to %q\n", file, channel)
		upload, err := u.FromPath(ctx, file)
		if err != nil {
			return fmt.Errorf("failed to upload %q: %w", file, err)
		}

		// Now we have uploaded file handle, sending it as styled message.
		// First, preparing message.
		document := message.UploadedDocument(upload,
			html.String(nil, `Upload: <b>from vaultDump</b>`),
		)
		document.MIME("application/sql").Filename(file)

		// Resolving target. Can be telephone number or @nickname of user,
		// group or channel.
		target := sender.Resolve(channel)

		// Sending message with media.
		fmt.Printf("Sending vaultPass dump %q to %q\n", file, channel)
		if _, err := target.Media(ctx, document); err != nil {
			return fmt.Errorf("failed to send file: %w", err)
		}
		return nil
	}

	// Run the upload.
	ctx := context.Background()
	return telegram.BotFromEnvironment(ctx, telegram.Options{NoUpdates: true}, nil, performUpload)
}

func main() {
	filename := fmt.Sprintf("vaultPass_dump_%s.sql", time.Now().Format("2006_01_02"))
	pgDump := exec.Command("pg_dumpall", "-w", "-d", os.Getenv("PGHOST"), "-f", filename)
	var stderr bytes.Buffer
	pgDump.Stderr = &stderr
	if err := pgDump.Run(); err != nil {
		log.Fatalf("Failed to run `pg_dumpall` command: %v: %s\n", err, stderr.String())
	}
	if err := uploadFile(filename, os.Getenv("TG_CHANNEL")); err != nil {
		log.Fatal(err)
	}
	// Remove local dump copy from the container.
	os.Remove(filename)
	// Wait one day for the next run.
	time.Sleep(24 * time.Hour)
}
