package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const bufferAPIURL = "https://api.buffer.com"

type Client struct {
	httpClient *http.Client
	token      string
	orgID      string
	maxQueue   int
}

func NewClient(token, orgID string, maxQueue int) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		orgID:      orgID,
		maxQueue:   maxQueue,
	}
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type Organization struct {
	ID string `json:"id"`
}

type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Post struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Status    string `json:"status"`
	ChannelID string `json:"channelId"`
}

func (c *Client) doGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(graphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bufferAPIURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(raw, &gqlResp); err != nil {
		return fmt.Errorf("decode response: %w; body=%s", err, string(raw))
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	fmt.Println(string(raw))

	if out == nil {
		return nil
	}

	return json.Unmarshal(raw, out)
}

func (c *Client) getChannels(ctx context.Context, orgID string) ([]Channel, error) {
	query := `
query GetChannels($organizationId: OrganizationId!) {
  channels(input: { organizationId: $organizationId }) {
    id
    name
    displayName
    service
    avatar
    isQueuePaused
  }
}`
	var resp struct {
		Data struct {
			Channels []Channel `json:"channels"`
		} `json:"data"`
	}
	if err := c.doGraphQL(ctx, query, map[string]any{
		"organizationId": orgID,
	}, &resp); err != nil {
		return nil, err
	}

	return resp.Data.Channels, nil
}
func (c *Client) getQueuePosts(ctx context.Context, orgID, channelID string) ([]Post, error) {
	query := `
query GetPosts($organizationId: OrganizationId!, $channelIds: [ChannelId!]) {
  posts(
    first: 100,
    input: {
      organizationId: $organizationId,
      filter: {
        status: [scheduled],
        channelIds: $channelIds
      }
    }
  ) {
    edges {
      node {
        id
        text
        status
        channelId
      }
    }
  }
}`
	var resp struct {
		Data struct {
			Posts struct {
				Edges []struct {
					Node Post `json:"node"`
				} `json:"edges"`
			} `json:"posts"`
		} `json:"data"`
	}
	if err := c.doGraphQL(ctx, query, map[string]any{
		"organizationId": orgID,
		"channelIds":     []string{channelID},
	}, &resp); err != nil {
		return nil, err
	}

	out := make([]Post, 0, len(resp.Data.Posts.Edges))
	for _, e := range resp.Data.Posts.Edges {
		out = append(out, e.Node)
	}
	return out, nil
}

func (c *Client) getDraftPosts(ctx context.Context, orgID, channelID string) ([]Post, error) {
	query := `
query GetDraftPosts($organizationId: OrganizationId!, $channelIds: [ChannelId!]) {
  posts(
    first: 100,
    input: {
      organizationId: $organizationId,
      filter: {
        status: [draft],
        channelIds: $channelIds
      }
    }
  ) {
    edges {
      node {
        id
        text
        status
        channelId
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`
	var resp struct {
		Data struct {
			Posts struct {
				Edges []struct {
					Node Post `json:"node"`
				} `json:"edges"`
			} `json:"posts"`
		} `json:"data"`
	}
	if err := c.doGraphQL(ctx, query, map[string]any{
		"organizationId": orgID,
		"channelIds":     []string{channelID},
	}, &resp); err != nil {
		return nil, err
	}

	drafts := make([]Post, 0, len(resp.Data.Posts.Edges))
	for _, e := range resp.Data.Posts.Edges {
		drafts = append(drafts, e.Node)
	}
	return drafts, nil
}

func (c *Client) pushDraftToQueue(ctx context.Context, draft Post) error {
	query := `
mutation CreatePost($input: CreatePostInput!) {
  createPost(input: $input) {
    ... on PostActionSuccess {
      post {
        id
        status
      }
    }
    ... on MutationError {
      message
    }
  }
}`

	var resp struct {
		Data struct {
			CreatePost struct {
				Post *struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"post,omitempty"`
				Message string `json:"message,omitempty"`
			} `json:"createPost"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := c.doGraphQL(ctx, query, map[string]any{
		"input": map[string]any{
			"text":           draft.Text,
			"channelId":      draft.ChannelID,
			"schedulingType": "automatic",
			"mode":           "addToQueue",
		},
	}, &resp); err != nil {
		return err
	}

	if len(resp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}

	if resp.Data.CreatePost.Message != "" {
		return fmt.Errorf("buffer createPost error: %s", resp.Data.CreatePost.Message)
	}

	if resp.Data.CreatePost.Post == nil {
		return fmt.Errorf("buffer createPost returned no post and no error")
	}

	return nil
}
