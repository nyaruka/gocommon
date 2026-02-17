package osearch

import (
	"context"
	"fmt"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/awsv2"

	awsx "github.com/nyaruka/gocommon/aws"
)

// NewClient creates a new OpenSearch API client. If access key and secret are provided, the client will use AWS SigV4
// request signing.
func NewClient(accessKey, secretKey, region, url string) (*opensearchapi.Client, error) {
	osConfig := opensearch.Config{
		Addresses: []string{url},
	}

	if accessKey != "" && secretKey != "" {
		cfg, err := awsx.NewConfig(accessKey, secretKey, region)
		if err != nil {
			return nil, err
		}

		// AWS OpenSearch Serverless uses "aoss" as the service name for signing
		svc := "es"
		if strings.Contains(url, ".aoss.") {
			svc = "aoss"
		}

		signer, err := requestsigner.NewSignerWithService(cfg, svc)
		if err != nil {
			return nil, fmt.Errorf("error creating opensearch request signer: %w", err)
		}

		osConfig.Signer = signer
	}

	return opensearchapi.NewClient(opensearchapi.Config{Client: osConfig})
}

// Test checks that the given indexes exist.
func Test(ctx context.Context, c *opensearchapi.Client, indexes ...string) error {
	for _, index := range indexes {
		resp, err := c.Indices.Exists(ctx, opensearchapi.IndicesExistsReq{
			Indices: []string{index},
		})
		if err != nil {
			return fmt.Errorf("error checking opensearch index %s: %w", index, err)
		}
		if resp.IsError() {
			return fmt.Errorf("opensearch index %s does not exist", index)
		}
	}

	return nil
}
