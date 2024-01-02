package japi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/jsonapi/gen/v1/jsonapi_pb"
	"github.com/pentops/jsonapi/structure"
	"github.com/pentops/jsonapi/swagger"
	"github.com/pentops/log.go/log"
	"github.com/rs/cors"
	"google.golang.org/protobuf/proto"
)

func NewRegistry(ctx context.Context, s3Client S3API, bucket string) (http.Handler, error) {

	bucketURL, err := url.Parse(bucket)
	if err != nil {
		return nil, err
	}

	if bucketURL.Scheme != "s3" {
		return nil, fmt.Errorf("bucket must be an s3:// url")
	}

	bucketName := bucketURL.Host
	if bucketName == "" {
		return nil, fmt.Errorf("bucket must be an s3:// url")
	}

	prefix := bucketURL.Path

	source := &S3Source{
		Bucket: bucketName,
		Prefix: prefix,
		Client: s3Client,
	}

	var handler http.Handler

	handler = &Handler{
		Source: source,
	}

	handler = http.StripPrefix("/registry/v1", handler)

	handler = cors.Default().Handler(handler)

	return handler, nil

}

type Handler struct {
	Source Source
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		log.Infof(ctx, "Expected five parts, found %v: %+v", len(parts), parts)
		http.NotFound(w, r)
		return
	}

	orgName := parts[1]
	imageName := parts[2]
	version := parts[3]
	format := parts[4]

	img, err := h.Source.GetImage(ctx, orgName, imageName, version)
	if err != nil {
		if errors.Is(err, ImageNotFoundError) {
			log.Infof(ctx, "Failed to get image: didn't find image for org/image/version %v/%v/%v", orgName, imageName, version)
			http.NotFound(w, r)
			return
		}

		log.WithError(ctx, err).Error("Failed to get image")
		http.Error(w, "Internal", http.StatusInternalServerError)
		return
	}

	switch format {
	case "swagger.json":
		swaggerContent, err := buildSwagger(ctx, img)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(swaggerContent) // nolint: errcheck

	case "jdef.json":
		jdefContent, err := buildJDef(ctx, img)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jdefContent) // nolint: errcheck

	case "image.bin":
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		imgBytes, err := proto.Marshal(img)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(imgBytes) // nolint: errcheck

	default:
		http.NotFound(w, r)
	}

}

func buildSwagger(ctx context.Context, img *jsonapi_pb.Image) ([]byte, error) {
	jdefDoc, err := structure.BuildFromImage(img)
	if err != nil {
		return nil, err
	}

	swaggerDoc, err := swagger.BuildSwagger(jdefDoc)
	if err != nil {
		return nil, err
	}

	asJson, err := json.Marshal(swaggerDoc)
	if err != nil {
		return nil, err
	}

	return asJson, nil
}

func buildJDef(ctx context.Context, img *jsonapi_pb.Image) ([]byte, error) {
	jdefDoc, err := structure.BuildFromImage(img)
	if err != nil {
		return nil, err
	}

	asJson, err := json.Marshal(jdefDoc)
	if err != nil {
		return nil, err
	}

	return asJson, nil
}

var ImageNotFoundError = fmt.Errorf("image not found")

type Source interface {
	GetImage(ctx context.Context, orgName, imageName string, version string) (*jsonapi_pb.Image, error)
}

type S3Source struct {
	Bucket string
	Prefix string
	Client S3API
}

type S3API interface {
	//PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func (s *S3Source) GetImage(ctx context.Context, orgName, imageName string, version string) (*jsonapi_pb.Image, error) {
	key := path.Join(s.Prefix, orgName, imageName, version, "image.bin")
	input := &s3.GetObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	}

	output, err := s.Client.GetObject(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			return nil, ImageNotFoundError
		}

		return nil, err
	}
	defer output.Body.Close()
	bodyBytes, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	img := &jsonapi_pb.Image{}
	if err := proto.Unmarshal(bodyBytes, img); err != nil {
		return nil, err
	}

	return img, nil
}
