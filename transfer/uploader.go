package transfer

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mrabhi2k3/telegofer/mtproto/connection"
	"github.com/mrabhi2k3/telegofer/tl/generated"
)

type Uploader struct {
	conn      *connection.Connection
	workers   int
	chunkSize int
}

func NewUploader(conn *connection.Connection, workers int, chunkSize int) *Uploader {
	cSize, _ := CalculateParts(1, chunkSize)
	if workers <= 0 {
		workers = 4
	}
	return &Uploader{
		conn:      conn,
		workers:   workers,
		chunkSize: cSize,
	}
}

type uploadPartJob struct {
	partIndex int
	data      []byte
}

func (u *Uploader) Upload(ctx context.Context, name string, r io.Reader, size int64, progress ProgressCallback) (generated.InputFileClass, error) {
	var fileIDBytes [8]byte
	if _, err := rand.Read(fileIDBytes[:]); err != nil {
		return nil, err
	}
	fileID := int64(binary.LittleEndian.Uint64(fileIDBytes[:]))

	chunkSize, totalParts := CalculateParts(size, u.chunkSize)
	isBig := size > BigFileThreshold

	var uploadedBytes atomic.Int64
	var md5Hasher = md5.New()

	jobs := make(chan uploadPartJob, u.workers)
	errChan := make(chan error, 1)

	var wg sync.WaitGroup

	for w := 0; w < u.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for job := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var invokeErr error
				if isBig {
					req := &generated.UploadSaveBigFilePart{
						FileId:         fileID,
						FilePart:       int32(job.partIndex),
						FileTotalParts: int32(totalParts),
						Bytes:          job.data,
					}
					_, invokeErr = u.conn.Invoke(ctx, req, 60*time.Second)
				} else {
					req := &generated.UploadSaveFilePart{
						FileId:   fileID,
						FilePart: int32(job.partIndex),
						Bytes:    job.data,
					}
					_, invokeErr = u.conn.Invoke(ctx, req, 60*time.Second)
				}

				if invokeErr != nil {
					select {
					case errChan <- invokeErr:
					default:
					}
					return
				}

				newTotal := uploadedBytes.Add(int64(len(job.data)))
				if progress != nil {
					progress(newTotal, size)
				}
			}
		}()
	}

	var readErr error
	partIdx := 0

	for {
		buf := make([]byte, chunkSize)
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			chunk := buf[:n]
			if !isBig {
				md5Hasher.Write(chunk)
			}

			select {
			case <-ctx.Done():
				readErr = ctx.Err()
				break
			case err := <-errChan:
				readErr = err
				break
			case jobs <- uploadPartJob{partIndex: partIdx, data: chunk}:
				partIdx++
			}
		}

		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				readErr = err
			}
			break
		}
	}

	close(jobs)
	wg.Wait()

	if readErr != nil {
		return nil, readErr
	}

	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	if isBig {
		return &generated.InputFileBig{
			Id:    fileID,
			Parts: int32(partIdx),
			Name:  name,
		}, nil
	}

	md5Checksum := fmt.Sprintf("%x", md5Hasher.Sum(nil))
	return &generated.InputFile{
		Id:          fileID,
		Parts:       int32(partIdx),
		Name:        name,
		Md5Checksum: md5Checksum,
	}, nil
}

