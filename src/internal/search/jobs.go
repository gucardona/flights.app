package search

import (
	"fmt"
	"math"
	"time"
)

const dateLayout = "2006-01-02"

// --- API request shapes from the frontend ---

type SearchRequest struct {
	APIKey      string `json:"api_key"`
	Origin      string `json:"origin"`
	Mode        string `json:"mode"` // "specific" | "range" | "month"
	Adults      int    `json:"adults"`
	TravelClass int    `json:"travel_class"`
	Currency    string `json:"currency"`

	// mode=specific
	Destinations []string `json:"destinations"` // one or many IATA codes
	OutboundDate string   `json:"outbound_date"`
	ReturnDate   string   `json:"return_date"`

	// mode=range
	OutboundFrom string `json:"outbound_from"`
	OutboundTo   string `json:"outbound_to"`
	ReturnFrom   string `json:"return_from"`
	ReturnTo     string `json:"return_to"`
	MaxCombos    int    `json:"max_combos"`

	// mode=month
	OutboundMonths []int `json:"outbound_months"` // 1-12
	ReturnMonths   []int `json:"return_months"`
	Year           int   `json:"year"`
	ReturnYear     int   `json:"return_year"` // if return is in next year
	SamplesPerMonth int  `json:"samples_per_month"`
}

func (r *SearchRequest) defaults() {
	if r.Adults == 0 {
		r.Adults = 1
	}
	if r.TravelClass == 0 {
		r.TravelClass = 1
	}
	if r.Currency == "" {
		r.Currency = "USD"
	}
	if r.MaxCombos == 0 {
		r.MaxCombos = 8
	}
	if r.SamplesPerMonth == 0 {
		r.SamplesPerMonth = 4
	}
}

func (r *SearchRequest) Validate() error {
	if r.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if r.Origin == "" {
		return fmt.Errorf("origin is required")
	}
	if len(r.Destinations) == 0 {
		return fmt.Errorf("at least one destination is required")
	}
	switch r.Mode {
	case "specific", "range", "month":
	default:
		return fmt.Errorf("mode must be 'specific', 'range', or 'month'")
	}
	return nil
}

// BuildJobs converts a SearchRequest into a flat list of search Jobs.
func BuildJobs(req *SearchRequest) ([]Job, error) {
	req.defaults()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var jobs []Job

	switch req.Mode {
	case "specific":
		for _, dest := range req.Destinations {
			jobs = append(jobs, Job{
				Origin:       req.Origin,
				Destination:  dest,
				OutboundDate: req.OutboundDate,
				ReturnDate:   req.ReturnDate,
				Adults:       req.Adults,
				TravelClass:  req.TravelClass,
				Currency:     req.Currency,
			})
		}

	case "range":
		outDates, err := datesInRange(req.OutboundFrom, req.OutboundTo, req.MaxCombos/len(req.Destinations)+1)
		if err != nil {
			return nil, fmt.Errorf("outbound range: %w", err)
		}
		var retDates []string
		if req.ReturnFrom != "" && req.ReturnTo != "" {
			retDates, err = datesInRange(req.ReturnFrom, req.ReturnTo, req.MaxCombos/len(req.Destinations)+1)
			if err != nil {
				return nil, fmt.Errorf("return range: %w", err)
			}
		}
		for _, dest := range req.Destinations {
			for i, od := range outDates {
				rd := ""
				if len(retDates) > 0 {
					rd = retDates[min(i, len(retDates)-1)]
				}
				jobs = append(jobs, Job{
					Origin:       req.Origin,
					Destination:  dest,
					OutboundDate: od,
					ReturnDate:   rd,
					Adults:       req.Adults,
					TravelClass:  req.TravelClass,
					Currency:     req.Currency,
				})
			}
		}
		if len(jobs) > req.MaxCombos {
			jobs = jobs[:req.MaxCombos]
		}

	case "month":
		if len(req.OutboundMonths) == 0 {
			return nil, fmt.Errorf("outbound_months is required for month mode")
		}
		outDates := datesForMonths(req.OutboundMonths, req.Year, req.SamplesPerMonth)
		var retDates []string
		if len(req.ReturnMonths) > 0 {
			ry := req.Year
			if req.ReturnYear > 0 {
				ry = req.ReturnYear
			}
			retDates = datesForMonths(req.ReturnMonths, ry, req.SamplesPerMonth)
		}
		for _, dest := range req.Destinations {
			for i, od := range outDates {
				rd := ""
				if len(retDates) > 0 {
					rd = retDates[min(i, len(retDates)-1)]
				}
				jobs = append(jobs, Job{
					Origin:       req.Origin,
					Destination:  dest,
					OutboundDate: od,
					ReturnDate:   rd,
					Adults:       req.Adults,
					TravelClass:  req.TravelClass,
					Currency:     req.Currency,
				})
			}
		}
	}

	return jobs, nil
}

// datesInRange returns up to maxCount evenly-spaced dates between from and to.
func datesInRange(from, to string, maxCount int) ([]string, error) {
	start, err := time.Parse(dateLayout, from)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", from, err)
	}
	end, err := time.Parse(dateLayout, to)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", to, err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("%q is before %q", to, from)
	}

	totalDays := int(end.Sub(start).Hours() / 24)
	step := 1
	if totalDays > maxCount {
		step = int(math.Ceil(float64(totalDays) / float64(maxCount)))
	}

	var dates []string
	for d := start; !d.After(end) && len(dates) < maxCount; d = d.AddDate(0, 0, step) {
		dates = append(dates, d.Format(dateLayout))
	}
	return dates, nil
}

// datesForMonths returns sampled dates across the given months.
func datesForMonths(months []int, year, samplesPerMonth int) []string {
	var dates []string
	for _, m := range months {
		daysInMonth := time.Date(year, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Day()
		step := daysInMonth / (samplesPerMonth + 1)
		if step < 1 {
			step = 1
		}
		for s := 1; s <= samplesPerMonth; s++ {
			day := min(s*step, daysInMonth)
			dates = append(dates, fmt.Sprintf("%04d-%02d-%02d", year, m, day))
		}
	}
	return dates
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
