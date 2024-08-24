package cloud

import (
	"context"

	"github.com/OmGuptaIND/env"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type AwsClientOptions struct{}

type AwsClient struct {
	ctx        context.Context
	bucketName string
	s3Client   *s3.S3
	*AwsClientOptions
}

// AwsClient handles the connection with AWS S3 Bucket of the recording to the cloud.
func NewAwsClient(ctx context.Context, opts *AwsClientOptions) (CloudClient, error) {
	bucketConfig := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(env.GetBucketKeyId(), env.GetBucketAppKey(), ""),
		Endpoint:         aws.String("https://s3.us-west-002.backblazeb2.com"),
		Region:           aws.String("eu-central-003"),
		S3ForcePathStyle: aws.Bool(true),
	}

	awsSessions, err := session.NewSession(bucketConfig)

	if err != nil {
		return nil, err
	}

	s3Client := s3.New(awsSessions)

	awsClient := &AwsClient{
		context.WithoutCancel(ctx),
		env.GetBucketName(),
		s3Client,
		opts,
	}

	return awsClient, nil
}
