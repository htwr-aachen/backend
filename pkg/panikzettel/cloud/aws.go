package cloud

import (
	"context"

	"github.com/htwr-aachen/backend/pkg/config"
	"gocloud.dev/blob"
)

type AWSCloudClient struct {
	bucket *blob.Bucket
}

func NewAWS(ctx context.Context, cfg *config.Config) (*AWSCloudClient, error) {
	return nil, nil
}

func (c *AWSCloudClient) Bucket() *blob.Bucket {
	return c.bucket
}

func (c *AWSCloudClient) Close() {
}
