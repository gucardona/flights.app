package search

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"flights/src/internal/serpapi"
)

// --- Public result types (returned to the frontend) ---

type FlightResult struct {
	Origin      string  `json:"origin"`
	Destination string  `json:"destination"`
	OutboundDate string `json:"outbound_date"`
	ReturnDate   string `json:"return_date,omitempty"`

	Price         float64 `json:"price"`
	TotalDuration int     `json:"total_duration"` // minutes
	Stops         int     `json:"stops"`
	Airline       string  `json:"airline"`
	FlightNumbers string  `json:"flight_numbers"`
	DepTime       string  `json:"dep_time"`
	ArrTime       string  `json:"arr_time"`
	DepIATA       string  `json:"dep_iata"`
	ArrIATA       string  `json:"arr_iata"`

	PriceLevel  string  `json:"price_level,omitempty"`
	LowestSeen  float64 `json:"lowest_seen,omitempty"`
	IsBest      bool    `json:"is_best"`
}

type SearchResults struct {
	Results    []FlightResult `json:"results"`
	TotalJobs  int            `json:"total_jobs"`
	Successful int            `json:"successful"`
	Errors     []string       `json:"errors,omitempty"`
}

// --- Job definition ---

type Job struct {
	Origin       string
	Destination  string
	OutboundDate string
	ReturnDate   string
	Adults       int
	TravelClass  int
	Currency     string
}

// --- Service ---

type Service struct {
	client      *serpapi.Client
	concurrency int
	delay       time.Duration
}

func NewService(apiKey string) *Service {
	return &Service{
		client:      serpapi.NewClient(apiKey),
		concurrency: 3,              // max parallel SerpAPI requests
		delay:       400 * time.Millisecond, // be polite to the API
	}
}

// Run executes all jobs concurrently (bounded by concurrency limit).
func (s *Service) Run(jobs []Job) *SearchResults {
	type jobResult struct {
		result *FlightResult
		err    error
	}

	sem := make(chan struct{}, s.concurrency)
	results := make(chan jobResult, len(jobs))
	var wg sync.WaitGroup

	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			time.Sleep(s.delay)

			fr, err := s.searchOne(job)
			results <- jobResult{fr, err}
		}()
	}

	wg.Wait()
	close(results)

	out := &SearchResults{TotalJobs: len(jobs)}
	for r := range results {
		if r.err != nil {
			out.Errors = append(out.Errors, r.err.Error())
			continue
		}
		if r.result != nil {
			out.Results = append(out.Results, *r.result)
			out.Successful++
		}
	}

	// Sort by price ascending
	sort.Slice(out.Results, func(i, j int) bool {
		return out.Results[i].Price < out.Results[j].Price
	})

	// Mark best
	if len(out.Results) > 0 {
		bestPrice := out.Results[0].Price
		for i := range out.Results {
			if out.Results[i].Price == bestPrice {
				out.Results[i].IsBest = true
			}
		}
	}

	return out
}

func (s *Service) searchOne(job Job) (*FlightResult, error) {
	req := serpapi.FlightRequest{
		Origin:       job.Origin,
		Destination:  job.Destination,
		OutboundDate: job.OutboundDate,
		ReturnDate:   job.ReturnDate,
		Adults:       job.Adults,
		TravelClass:  job.TravelClass,
		Currency:     job.Currency,
	}

	resp, err := s.client.Search(req)
	if err != nil {
		return nil, fmt.Errorf("[%s→%s %s]: %w", job.Origin, job.Destination, job.OutboundDate, err)
	}

	flights := resp.BestFlights
	if len(flights) == 0 {
		flights = resp.OtherFlights
	}
	if len(flights) == 0 {
		return nil, nil // no flights found, not an error
	}

	best := flights[0]
	legs := best.Flights
	if len(legs) == 0 {
		return nil, nil
	}

	firstLeg := legs[0]
	lastLeg := legs[len(legs)-1]

	// Collect all flight numbers
	var fnums string
	for i, l := range legs {
		if i > 0 {
			fnums += " / "
		}
		fnums += l.FlightNumber
	}

	fr := &FlightResult{
		Origin:        job.Origin,
		Destination:   job.Destination,
		OutboundDate:  job.OutboundDate,
		ReturnDate:    job.ReturnDate,
		Price:         best.Price,
		TotalDuration: best.TotalDuration,
		Stops:         len(legs) - 1,
		Airline:       firstLeg.Airline,
		FlightNumbers: fnums,
		DepTime:       firstLeg.DepartureAirport.Time,
		ArrTime:       lastLeg.ArrivalAirport.Time,
		DepIATA:       firstLeg.DepartureAirport.ID,
		ArrIATA:       lastLeg.ArrivalAirport.ID,
	}

	if resp.PriceInsights != nil {
		fr.PriceLevel = resp.PriceInsights.PriceLevel
		fr.LowestSeen = float64(resp.PriceInsights.LowestPrice)
	}

	return fr, nil
}
