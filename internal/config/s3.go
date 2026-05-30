package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultS3Region         = "us-east-1"
	defaultS3ForcePathStyle = true
)

func s3BlobConfigFromEnv(blobBackend string) (S3BlobConfig, error) {
	cfg := S3BlobConfig{
		Endpoint:        strings.TrimSpace(os.Getenv("SAFE_S3_ENDPOINT")),
		Region:          strings.TrimSpace(envOrDefault("SAFE_S3_REGION", defaultS3Region)),
		Bucket:          strings.TrimSpace(os.Getenv("SAFE_S3_BUCKET")),
		Prefix:          strings.TrimSpace(os.Getenv("SAFE_S3_PREFIX")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("SAFE_S3_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("SAFE_S3_SECRET_ACCESS_KEY")),
		SessionToken:    strings.TrimSpace(os.Getenv("SAFE_S3_SESSION_TOKEN")),
		ForcePathStyle:  defaultS3ForcePathStyle,
	}

	if blobBackend != BlobBackendS3 {
		return cfg, nil
	}

	forcePathStyle, err := boolFromEnv("SAFE_S3_FORCE_PATH_STYLE", defaultS3ForcePathStyle)
	if err != nil {
		return S3BlobConfig{}, err
	}
	cfg.ForcePathStyle = forcePathStyle

	if cfg.Endpoint == "" {
		return S3BlobConfig{}, fmt.Errorf("parse SAFE_S3_ENDPOINT: required when SAFE_BLOB_BACKEND=s3")
	}
	if cfg.Region == "" {
		return S3BlobConfig{}, fmt.Errorf("parse SAFE_S3_REGION: required when SAFE_BLOB_BACKEND=s3")
	}
	if cfg.Bucket == "" {
		return S3BlobConfig{}, fmt.Errorf("parse SAFE_S3_BUCKET: required when SAFE_BLOB_BACKEND=s3")
	}
	if cfg.AccessKeyID == "" {
		return S3BlobConfig{}, fmt.Errorf("parse SAFE_S3_ACCESS_KEY_ID: required when SAFE_BLOB_BACKEND=s3")
	}
	if cfg.SecretAccessKey == "" {
		return S3BlobConfig{}, fmt.Errorf("parse SAFE_S3_SECRET_ACCESS_KEY: required when SAFE_BLOB_BACKEND=s3")
	}
	return cfg, nil
}

func boolFromEnv(name string, fallback bool) (bool, error) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, fmt.Errorf("parse %s: empty boolean", name)
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: invalid boolean", name)
	}
	return parsed, nil
}
