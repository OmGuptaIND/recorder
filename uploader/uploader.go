package uploader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/OmGuptaIND/cloud"
	"github.com/OmGuptaIND/config"
)

type Uploader struct {
	id     string
	ctx    context.Context
	closed atomic.Bool
	wg     *sync.WaitGroup

	recordingId *string
	storagePath string

	reader *bufio.Reader
	client cloud.CloudClient

	partNumber int

	completedMtx   *sync.Mutex
	completedParts []*cloud.CloudUploadPartReponse
	buffer         []byte
}

// NewUploader creates a new Uploader instance.
func NewUploader(ctx context.Context, reader *bufio.Reader, recordingId *string) (*Uploader, error) {
	uploadCtx := context.WithoutCancel(ctx)

	cloudClient := cloud.GetClient(&uploadCtx)

	storagePath := fmt.Sprintf("recording_%s.mp4", *recordingId)

	uploaderId, err := cloudClient.CreateMultipartUpload(&storagePath)

	if err != nil {
		return nil, err
	}

	uploader := &Uploader{
		ctx:         uploadCtx,
		id:          *uploaderId,
		recordingId: recordingId,
		storagePath: storagePath,
		closed:      atomic.Bool{},

		wg: &sync.WaitGroup{},

		client: cloudClient,
		reader: reader,

		partNumber: 1,

		completedMtx:   &sync.Mutex{},
		completedParts: make([]*cloud.CloudUploadPartReponse, 0),

		buffer: make([]byte, config.MAX_BUFFER_SIZE),
	}

	return uploader, nil
}

// GetContext returns the context of the Uploader.
func (u *Uploader) GetContext() context.Context {
	return u.ctx
}

// GetID returns the ID of the Uploader.
func (u *Uploader) GetID() string {
	return u.id
}

// GetObjectKey returns the object key.
func (u *Uploader) GetObjectKey() *string {
	key := u.storagePath

	return &key
}

// Wait waits for the Uploader to finish.
func (u *Uploader) Wait() {
	u.wg.Wait()
}

// AddPart adds a part to the Uploader.
func (u *Uploader) addCompletedPart(part *cloud.CloudUploadPartReponse) {
	u.completedMtx.Lock()
	defer u.completedMtx.Unlock()

	u.completedParts = append(u.completedParts, part)
}

// Start starts the Uploader.
func (u *Uploader) Start() error {
	u.wg.Add(1)
	defer func() {
		log.Println("Start Uploader is done, Getting out of Start Uploader")
		u.wg.Done()
	}()

	if u.recordingId == nil {
		log.Println("No recording found to upload.")
		return fmt.Errorf("no recording found to upload: %s", u.GetID())
	}

	bytesRead := 0

	for {
		n, err := u.reader.Read(u.buffer[bytesRead:])

		if err != nil && err != io.EOF && n == 0 {
			return fmt.Errorf("failed to read from reader: %v", err)
		}

		bytesRead += n

		if bytesRead >= int(config.MAX_BUFFER_SIZE) || (err == io.EOF && bytesRead > 0) {
			log.Println("New Part Started", u.partNumber, " Size", bytesRead, " Req_Size", config.MAX_BUFFER_SIZE)
			tempBuffer := make([]byte, bytesRead)
			copy(tempBuffer, u.buffer[:bytesRead])

			partInput := &cloud.CloudUploadPartInput{
				UploadId:    u.GetID(),
				StoragePath: u.GetObjectKey(),
				Buffer:      &tempBuffer,
				PartNumber:  u.partNumber,
			}

			u.wg.Add(1)
			go func() {
				defer u.wg.Done()
				part, err := u.uploadPart(partInput)

				if err != nil {
					log.Println("Failed to upload part", partInput.PartNumber, err)
					return
				}

				u.addCompletedPart(part)
			}()

			u.partNumber++
			bytesRead = 0
		}

		if err == io.EOF {
			log.Println("EOF reached")
			break
		}
	}

	return nil
}

// Stop stops the Uploader, buffers everything from the recording onto a temp buffer after which start a different goroutine to upload the last part.
// After all parts are uploaded, it completes the upload.
func (u *Uploader) Stop() (*cloud.CloudUploadPartCompleted, error) {
	log.Println("Stopping uploader...")

	if u.closed.Load() {
		log.Println("Uploader is already closed")
		return nil, fmt.Errorf("uploader is already closed")
	}

	u.wg.Wait()

	resp, err := u.completeUpload()

	if err != nil {
		log.Println("Failed to complete upload", err)
	}

	u.closed.Store(true)

	return resp, nil
}

// completeUpload completes the upload.
func (u *Uploader) completeUpload() (*cloud.CloudUploadPartCompleted, error) {
	log.Println("Completing upload...", len(u.completedParts))

	resp, err := u.client.CompletePartUpload(&cloud.CloudUploadPartInput{
		UploadId:    u.GetID(),
		StoragePath: u.GetObjectKey(),
		Parts:       &u.completedParts,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to complete multipart upload: %v", err)
	}

	log.Println("Multipart upload completed", u.GetID(), " Completed Parts Count", len(u.completedParts))

	return &cloud.CloudUploadPartCompleted{
		Recording_Url: resp.Recording_Url,
	}, nil
}

// Upload uploads the recording to the cloud.
func (u *Uploader) uploadPart(input *cloud.CloudUploadPartInput) (*cloud.CloudUploadPartReponse, error) {
	log.Println("Uploading part to cloud...", input.PartNumber)

	partResp, err := u.client.UploadPart(input)

	if err != nil {
		return nil, err
	}

	log.Println("Part uploaded successfully", input.PartNumber, " ETag", partResp.ETag)

	return partResp, nil
}
