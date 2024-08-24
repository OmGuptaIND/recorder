package cloud

import (
	"context"
	"os"

	"github.com/OmGuptaIND/env"
	"github.com/aws/aws-sdk-go/aws"
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
