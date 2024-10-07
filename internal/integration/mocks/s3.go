package mocks

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/o5-deploy-aws/internal/aws/aws_cf"
)

type S3 struct {
	aws_cf.S3API
	files map[string][]byte
}

func NewS3() *S3 {
	return &S3{
		files: map[string][]byte{},
	}
}

func (s3m *S3) MockGet(bucket string, key string) ([]byte, bool) {
	fullPath := fmt.Sprintf("s3://%s/%s", bucket, key)
	val, ok := s3m.files[fullPath]
	return val, ok
}

func (s3m *S3) MockGetHTTP(uri string) ([]byte, bool) {
	fullPath := "s3://" + strings.TrimPrefix(uri, "https://s3.us-east-1.amazonaws.com/")
	val, ok := s3m.files[fullPath]
	return val, ok
}

func (s3m *S3) PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	fullPath := fmt.Sprintf("s3://%s/%s", *input.Bucket, *input.Key)
	bodyBytes, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}

	s3m.files[fullPath] = bodyBytes
	return &s3.PutObjectOutput{}, nil
}

func (s3m *S3) GetBucketLocation(ctx context.Context, input *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return &s3.GetBucketLocationOutput{
		LocationConstraint: "us-east-1",
	}, nil
}
