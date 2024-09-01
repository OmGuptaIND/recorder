package cloud

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/OmGuptaIND/env"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type AwsClientOptions struct{}

type AwsClient struct {
	ctx        context.Context
	bucketName string
	s3Client   *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	*AwsClientOptions
}

// AwsClient handles the connection with AWS S3 Bucket of the recording to the cloud.
func NewAwsClient(ctx context.Context, opts *AwsClientOptions) (CloudClient, error) {
	bucketConfig := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(env.GetBucketKeyId(), env.GetBucketAppKey(), ""),
		Endpoint:         aws.String(env.GetBucketEndpoint()),
		Region:           aws.String(env.GetBucketRegion()),
		S3ForcePathStyle: aws.Bool(true),
		Retryer: client.DefaultRetryer{
			NumMaxRetries: 5,
			MinRetryDelay: 2 * time.Second,
			MaxRetryDelay: 30 * time.Second,
		},
	}

	awsSessions, err := session.NewSession(bucketConfig)

	if err != nil {
		return nil, err
	}

	s3Client := s3.New(awsSessions)
	uploader := s3manager.NewUploader(awsSessions)
	downloader := s3manager.NewDownloader(awsSessions)

	awsClient := &AwsClient{
		context.WithoutCancel(ctx),
		env.GetBucketName(),
		s3Client,
		uploader,
		downloader,
		opts,
	}

	return awsClient, nil
}

// CreateMultipartUpload creates a new multipart upload session for the file.
func (a *AwsClient) CreateMultipartUpload(storagePath *string) (*string, error) {
	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(a.bucketName),
		Key:         aws.String(*storagePath),
		ContentType: aws.String("video/mp4"),
	}

	result, err := a.s3Client.CreateMultipartUpload(input)

	if err != nil {
		return nil, err
	}

	uploadId := result.UploadId

	return uploadId, nil
}

// UploadPart uploads a part of the file to the cloud.
func (a *AwsClient) UploadPart(input *CloudUploadPartInput) (*CloudUploadPartReponse, error) {
	partInput := &s3.UploadPartInput{
		Body:       bytes.NewReader(*input.Buffer),
		Bucket:     aws.String(a.bucketName),
		Key:        aws.String(*input.StoragePath),
		PartNumber: aws.Int64(int64(input.PartNumber)),
		UploadId:   aws.String(input.UploadId),
	}

	partResp, err := a.s3Client.UploadPart(partInput)

	if err != nil {
		return nil, fmt.Errorf("failed to upload part %d: %v", *partInput.PartNumber, err)
	}

	return &CloudUploadPartReponse{
		ETag:       partResp.ETag,
		PartNumber: partInput.PartNumber,
	}, nil
}

// CompleteMultipartUpload completes the multipart upload session for the file.
func (a *AwsClient) CompletePartUpload(input *CloudUploadPartInput) (*CloudUploadPartCompleted, error) {
	completedParts := make([]*s3.CompletedPart, 0)

	for _, part := range *input.Parts {
		completedParts = append(completedParts, &s3.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		})
	}

	_, err := a.s3Client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		UploadId: aws.String(input.UploadId),
		Bucket:   aws.String(a.bucketName),
		Key:      input.StoragePath,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to complete multipart upload: %v", err)
	}

	recordingUrl := fmt.Sprintf("https://%s.%s/%s", a.bucketName, env.GetBucketEndpoint(), *input.StoragePath)

	return &CloudUploadPartCompleted{
		Recording_Url: &recordingUrl,
	}, nil
}

// UploadFile uploads the file to the cloud, using AWS Uploader which streams the file to the cloud.
func (a *AwsClient) UploadFile(fileName *string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = a.uploader.Upload(&s3manager.UploadInput{
		Body:   file,
		Bucket: &a.bucketName,
		Key:    fileName,
	})

	return err
}

// DownloadFile downloads the file from the cloud, using AWS Downloader which streams the file from the cloud.
func (a *AwsClient) DownloadFile(fileName *string, downloadPath string) error {
	file, err := os.Create(downloadPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = a.downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(a.bucketName),
		Key:    fileName,
	})

	return err
}
