package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

const (
	// GraphQLEndpoint is the Tibber GraphQL API URL
	GraphQLEndpoint = "https://api.tibber.com/v1-beta/gql"

	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second
)

// Client handles communication with Tibber API
type Client struct {
	token      string
	httpClient *http.Client
	endpoint   string
}

// NewClient creates a new Tibber API client
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		endpoint: GraphQLEndpoint,
	}
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string `json:"message"`
}

// execute sends a GraphQL request and returns the raw response
func (c *Client) execute(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

// GetHomes fetches all homes for the authenticated user
func (c *Client) GetHomes(ctx context.Context) ([]models.HomeResponse, error) {
	data, err := c.execute(ctx, QueryHomes, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Viewer struct {
			Homes []models.HomeResponse `json:"homes"`
		} `json:"viewer"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse homes: %w", err)
	}

	return result.Viewer.Homes, nil
}

// GetPrices fetches price information for a specific home
func (c *Client) GetPrices(ctx context.Context, homeID string) (*models.PriceInfo, error) {
	variables := map[string]interface{}{}

	data, err := c.execute(ctx, QueryPrices, variables)
	if err != nil {
		return nil, err
	}

	var result struct {
		Viewer struct {
			Homes []struct {
				ID                  string `json:"id"`
				CurrentSubscription *struct {
					PriceInfo *models.PriceInfo `json:"priceInfo"`
				} `json:"currentSubscription"`
			} `json:"homes"`
		} `json:"viewer"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prices: %w", err)
	}

	// Find the matching home or use first one
	for _, home := range result.Viewer.Homes {
		if homeID == "" || home.ID == homeID {
			if home.CurrentSubscription != nil && home.CurrentSubscription.PriceInfo != nil {
				return home.CurrentSubscription.PriceInfo, nil
			}
		}
	}

	return nil, fmt.Errorf("no price information found")
}
