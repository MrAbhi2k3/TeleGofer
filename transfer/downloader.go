package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mrabhi2k3/telegofer/mtproto/connection"
	"github.com/mrabhi2k3/telegofer/tl/decoder"
	"github.com/mrabhi2k3/telegofer/tl/generated"
)

type Downloader struct {
	conn      *connection.Connection
	workers   int
	chunkSize int
}

func NewDownloader(conn *connection.Connection, workers int, chunkSize int) *Downloader {
	cSize, _ := CalculateParts(1, chunkSize)
	if workers <= 0 {
		workers = 4
	}
	return &Downloader{
		conn:      conn,
		workers:   workers,
		chunkSize: cSize,
	}
}

func (d *Downloader) Download(ctx context.Context, location generated.InputFileLocationClass, totalSize int64, w io.Writer, progress ProgressCallback) error {
	var offset int64 = 0
	var downloaded int64 = 0
	limit := int32(d.chunkSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req := &generated.UploadGetFile{
			Precise:      true,
			CdnSupported: true,
			Location:     location,
			Offset:       offset,
			Limit:        limit,
		}

		respRaw, err := d.conn.Invoke(ctx, req, 60*time.Second)
		if err != nil {
			return fmt.Errorf("download: failed at offset %d: %w", offset, err)
		}

		fileRes, err := generated.DecodeUploadFileClass(decoder.New(respRaw))
		if err != nil {
			return fmt.Errorf("download: failed to decode file response: %w", err)
		}

		switch f := fileRes.(type) {
		case *generated.UploadFile:
			if len(f.Bytes) == 0 {
				return nil
			}

			if _, err := w.Write(f.Bytes); err != nil {
				return fmt.Errorf("download: failed to write to destination: %w", err)
			}

			n := int64(len(f.Bytes))
			offset += n
			downloaded += n

			if progress != nil {
				progress(downloaded, totalSize)
			}

			if int32(len(f.Bytes)) < limit {
				return nil
			}

		case *generated.UploadFileCdnRedirect:
			req.CdnSupported = false
			fallbackRaw, err := d.conn.Invoke(ctx, req, 60*time.Second)
			if err != nil {
				return fmt.Errorf("download: CDN fallback failed: %w", err)
			}
			fbRes, err := generated.DecodeUploadFileClass(decoder.New(fallbackRaw))
			if err != nil {
				return err
			}
			regularFile, ok := fbRes.(*generated.UploadFile)
			if !ok || len(regularFile.Bytes) == 0 {
				return nil
			}
			if _, err := w.Write(regularFile.Bytes); err != nil {
				return err
			}
			n := int64(len(regularFile.Bytes))
			offset += n
			downloaded += n
			if progress != nil {
				progress(downloaded, totalSize)
			}
			if int32(len(regularFile.Bytes)) < limit {
				return nil
			}

		default:
			return errors.New("download: unexpected file response type")
		}
	}
}

