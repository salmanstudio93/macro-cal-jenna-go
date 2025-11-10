package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/MacroPath/macro-path-backend/services/mealgen-service/models"
)

type FoodService struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewFoodService(apiKey string) *FoodService {
	return &FoodService{
		apiKey:  apiKey,
		baseURL: "https://api.studio93.io/food/search",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (fs *FoodService) SearchFood(foodName string) (*models.FoodAPIResult, error) {
	// Build the request URL with query parameters
	reqURL, err := url.Parse(fs.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Add query parameters with default values
	params := reqURL.Query()
	params.Add("food_name", foodName)
	params.Add("page_number", "0")  // Default to first page
	params.Add("max_results", "20") // Default to 20 results
	reqURL.RawQuery = params.Encode()

	// Create the HTTP request
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+fs.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := fs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var apiResponse struct {
		Message string               `json:"message"`
		Data    models.FoodAPIResult `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &apiResponse.Data, nil
}

// SearchFoodByBarcode searches for food items by barcode
func (fs *FoodService) SearchFoodByBarcode(barcode string, pageNumber int, maxResults int) (*models.FoodAPIResult, error) {
	// Build the request URL with query parameters
	reqURL, err := url.Parse(fs.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Add query parameters
	params := reqURL.Query()
	params.Add("barcode", barcode)
	params.Add("page_number", strconv.Itoa(pageNumber))
	params.Add("max_results", strconv.Itoa(maxResults))
	reqURL.RawQuery = params.Encode()

	// Create the HTTP request
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+fs.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := fs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var apiResponse struct {
		Message string               `json:"message"`
		Data    models.FoodAPIResult `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &apiResponse.Data, nil
}
