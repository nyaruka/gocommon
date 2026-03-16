package elastic_test

import (
	"os"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nyaruka/gocommon/elastic"
)

const elasticURL = "http://elastic:9200"

var testClient *elasticsearch.Client

func TestMain(m *testing.M) {
	var err error
	testClient, err = elastic.NewClient(elasticURL)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
