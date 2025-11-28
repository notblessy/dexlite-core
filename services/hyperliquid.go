package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	HYPERLIQUID_API_URL = "https://api.hyperliquid.xyz/info"
)

type HyperLiquidClient struct {
	client  *http.Client
	baseURL string
}

type MarketDataResponse struct {
	Universe []UniverseItem `json:"universe"`
}

type UniverseItem struct {
	Name  string `json:"name"`
	SzDec int    `json:"szDec"`
}

type PerpInfo struct {
	Funding      string   `json:"funding"`
	ImpactPxs    []string `json:"impactPxs"`
	MarkPx       string   `json:"markPx"`
	MidPx        string   `json:"midPx"`
	OpenInterest string   `json:"openInterest"`
	OraclePx     string   `json:"oraclePx"`
	Premiums     []string `json:"premiums"`
	Volume24h    string   `json:"volume24h"`
}

type Meta struct {
	Universe []UniverseItem `json:"universe"`
}

// AllMidsResponse represents the response from Hyperliquid allMids endpoint
// The actual response structure may vary, so we'll handle multiple formats
type AllMidsResponse struct {
	// Format 1: Direct mids map (like WebSocket response)
	Mids map[string]string `json:"mids,omitempty"`

	// Format 2: Nested structure with meta and data
	Meta Meta                `json:"meta,omitempty"`
	Data map[string]PerpInfo `json:"data,omitempty"`
}

// WrappedAllMidsResponse handles responses wrapped in a "data" field (like WebSocket format)
type WrappedAllMidsResponse struct {
	Data struct {
		Mids map[string]string `json:"mids,omitempty"`
	} `json:"data,omitempty"`
}

func NewHyperLiquidClient() *HyperLiquidClient {
	return &HyperLiquidClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: HYPERLIQUID_API_URL,
	}
}

// GetPrice fetches the current price for a given coin symbol
func (c *HyperLiquidClient) GetPrice(coin string) (float64, error) {
	// HyperLiquid uses coin names like "BTC", "ETH", etc.
	// We need to get all mids and find the one matching our coin

	// HyperLiquid API expects POST with body containing the request
	body := map[string]interface{}{
		"type": "allMids",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read the response body first to allow multiple parsing attempts
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try Format 1: Direct map[string]string (most common format for allMids)
	var directMids map[string]string
	if err := json.Unmarshal(bodyBytes, &directMids); err == nil && len(directMids) > 0 {
		// Try exact match first
		if priceStr, exists := directMids[coin]; exists {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
			}
			return price, nil
		}
		// Try case-insensitive match
		coinUpper := strings.ToUpper(coin)
		for key, priceStr := range directMids {
			if strings.ToUpper(key) == coinUpper {
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
				}
				return price, nil
			}
		}
	}

	// Try Format 2: Wrapped response (like WebSocket format: { data: { mids: {...} } })
	var wrappedResponse WrappedAllMidsResponse
	if err := json.Unmarshal(bodyBytes, &wrappedResponse); err == nil && wrappedResponse.Data.Mids != nil {
		if priceStr, exists := wrappedResponse.Data.Mids[coin]; exists {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
			}
			return price, nil
		}
		// Try case-insensitive match
		coinUpper := strings.ToUpper(coin)
		for key, priceStr := range wrappedResponse.Data.Mids {
			if strings.ToUpper(key) == coinUpper {
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
				}
				return price, nil
			}
		}
	}

	// Try Format 3: Direct AllMidsResponse with mids
	var response AllMidsResponse
	if err := json.Unmarshal(bodyBytes, &response); err == nil {
		// Try Format 3a: Direct mids map
		if response.Mids != nil {
			if priceStr, exists := response.Mids[coin]; exists {
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
				}
				return price, nil
			}
			// Try case-insensitive match
			coinUpper := strings.ToUpper(coin)
			for key, priceStr := range response.Mids {
				if strings.ToUpper(key) == coinUpper {
					price, err := strconv.ParseFloat(priceStr, 64)
					if err != nil {
						return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
					}
					return price, nil
				}
			}
		}

		// Try Format 3b: Nested structure with PerpInfo
		if response.Data != nil {
			if perpInfo, exists := response.Data[coin]; exists {
				// Parse the midPx as float64
				price, err := strconv.ParseFloat(perpInfo.MidPx, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
				}
				return price, nil
			}
			// Try case-insensitive match
			coinUpper := strings.ToUpper(coin)
			for key, perpInfo := range response.Data {
				if strings.ToUpper(key) == coinUpper {
					price, err := strconv.ParseFloat(perpInfo.MidPx, 64)
					if err != nil {
						return 0, fmt.Errorf("failed to parse price for %s: %w", coin, err)
					}
					return price, nil
				}
			}
		}
	}

	// If all parsing attempts failed, try to get debug info
	var genericResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &genericResponse); err == nil {
		// Extract all available coin symbols for debugging
		availableCoins := getMapKeys(genericResponse)
		return 0, fmt.Errorf("coin %s not found in response. Available coins in response: %v", coin, availableCoins)
	}

	// Last resort: try to parse as array or other structure
	return 0, fmt.Errorf("coin %s not found. Response (first 500 chars): %s", coin, string(bodyBytes[:min(500, len(bodyBytes))]))
}

// getAvailableCoins extracts available coin symbols from the response for debugging
func getAvailableCoins(response *AllMidsResponse) []string {
	var coins []string

	if response.Mids != nil {
		for coin := range response.Mids {
			coins = append(coins, coin)
		}
	}

	if response.Data != nil {
		for coin := range response.Data {
			coins = append(coins, coin)
		}
	}

	return coins
}

// getMapKeys extracts keys from a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
