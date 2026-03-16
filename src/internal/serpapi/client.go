package serpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://serpapi.com/search.json"

type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// --- Request ---

type FlightRequest struct {
	Origin      string // IATA code
	Destination string // IATA code
	OutboundDate string // YYYY-MM-DD
	ReturnDate   string // YYYY-MM-DD (empty = one-way)
	Adults       int
	TravelClass  int    // 1=economy 2=premium 3=business 4=first
	Currency     string // BRL, USD, EUR, GBP
}

// --- Raw SerpAPI response types ---

type FlightLeg struct {
	DepartureAirport struct {
		Name string `json:"name"`
		ID   string `json:"id"`
		Time string `json:"time"`
	} `json:"departure_airport"`
	ArrivalAirport struct {
		Name string `json:"name"`
		ID   string `json:"id"`
		Time string `json:"time"`
	} `json:"arrival_airport"`
	Duration int    `json:"duration"`
	Airplane string `json:"airplane"`
	Airline  string `json:"airline"`
	FlightNumber string `json:"flight_number"`
}

type FlightOption struct {
	Flights       []FlightLeg `json:"flights"`
	TotalDuration int         `json:"total_duration"`
	Price         float64     `json:"price"`
	Type          string      `json:"type"`
	CarbonEmissions *struct {
		ThisFlight int `json:"this_flight"`
	} `json:"carbon_emissions"`
	BookingToken string `json:"booking_token"`
}

type PriceInsights struct {
	LowestPrice int    `json:"lowest_price"`
	PriceLevel  string `json:"price_level"`
	PriceHistory [][]interface{} `json:"price_history"`
}

type SerpResponse struct {
	BestFlights  []FlightOption `json:"best_flights"`
	OtherFlights []FlightOption `json:"other_flights"`
	PriceInsights *PriceInsights `json:"price_insights"`
	Error        string         `json:"error"`
}

// Search calls SerpAPI and returns the raw response.
func (c *Client) Search(req FlightRequest) (*SerpResponse, error) {
	params := url.Values{}
	params.Set("engine", "google_flights")
	params.Set("api_key", c.APIKey)
	params.Set("departure_id", req.Origin)
	params.Set("arrival_id", req.Destination)
	params.Set("outbound_date", req.OutboundDate)
	params.Set("hl", "en")
	params.Set("currency", req.Currency)
	params.Set("adults", fmt.Sprintf("%d", req.Adults))
	params.Set("travel_class", fmt.Sprintf("%d", req.TravelClass))

	if req.ReturnDate != "" {
		params.Set("return_date", req.ReturnDate)
		params.Set("type", "1") // round trip
	} else {
		params.Set("type", "2") // one way
	}

	resp, err := c.HTTPClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serpapi returned status %d", resp.StatusCode)
	}

	var result SerpResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("serpapi error: %s", result.Error)
	}

	return &result, nil
}
