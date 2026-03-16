package elastic_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nyaruka/gocommon/elastic"
)

const defaultElasticURL = "http://elastic:9200"

var testClient *elasticsearch.Client

func TestMain(m *testing.M) {
	url := os.Getenv("ELASTICSEARCH_URL")
	if url == "" {
		url = defaultElasticURL
	}

	// verify Elasticsearch is reachable
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("elastic: Elasticsearch not reachable at", url)
		os.Exit(1)
	}
	resp.Body.Close()

	testClient, err = elastic.NewClient(url)
	if err != nil {
		fmt.Println("elastic: error creating client:", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
