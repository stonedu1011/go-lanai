package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"io/ioutil"
)

// SearchResponse modeled after https://opensearch.org/docs/latest/opensearch/rest-api/search/#response-body
type SearchResponse[T any] struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	}
	Hits struct {
		MaxScore float64 `json:"max_score"`
		Total    struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Index  string  `json:"_index"`
			ID     string  `json:"_id"`
			Score  float64 `json:"_score"`
			Source T       `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (c RepoImpl[T]) Search(ctx context.Context, dest *[]T, body interface{}, o ...Option[opensearchapi.SearchRequest]) error {
	var buffer bytes.Buffer
	err := json.NewEncoder(&buffer).Encode(body)
	if err != nil {
		return fmt.Errorf("unable to encode mapping: %w", err)
	}
	o = append(o, Search.WithBody(&buffer))
	resp, err := c.client.Search(ctx, o...)
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("error status code: %d", resp.StatusCode)
	}
	var searchResp SearchResponse[T]
	err = json.Unmarshal(respBody, &searchResp)
	if err != nil {
		return err
	}
	retModel := make([]T, len(searchResp.Hits.Hits))
	for i, hits := range searchResp.Hits.Hits {
		retModel[i] = hits.Source
	}
	*dest = retModel
	return nil
}

func (c *OpenClientImpl) Search(ctx context.Context, o ...Option[opensearchapi.SearchRequest]) (*opensearchapi.Response, error) {
	options := make([]func(request *opensearchapi.SearchRequest), len(o))
	for i, v := range o {
		options[i] = v
	}
	//nolint:makezero // linter doesn't like us appending to a non-zero initialized length, however we've exhausted its length in the for-loop above
	options = append(options, Search.WithContext(ctx))
	return c.client.API.Search(options...)
}

// searchExt can be extended
//	func (s searchExt) WithSomething() func(request *opensearchapi.SearchRequest) {
//		return func(request *opensearchapi.SearchRequest) {
//		}
//	}
type searchExt struct {
	opensearchapi.Search
}

var Search = searchExt{}