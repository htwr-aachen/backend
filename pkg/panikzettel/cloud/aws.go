package cloud

import (
	"context"

	"github.com/htwr-aachen/backend/pkg/panikzettel/config"
	"gocloud.dev/blob"
)

type AWSCloudClient struct {
	bucket *blob.Bucket
}

func NewAWS(ctx context.Context, cfg config.CloudConfig) (*AWSCloudClient, error) {
	return nil, nil
}

func (c *AWSCloudClient) Bucket() *blob.Bucket {
	return c.bucket
}

func (c *AWSCloudClient) Close() {
}
