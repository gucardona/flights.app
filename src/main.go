package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"flights/src/internal/search"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	mux := http.NewServeMux()

	// Serve static frontend files from ./src/web
	mux.Handle("/", http.FileServer(http.Dir("./src/web")))

	// Search endpoint
	mux.HandleFunc("/api/search", handleSearch)

	// Regions helper endpoint
	mux.HandleFunc("/api/regions", handleRegions)

	log.Printf("FlightRadar running on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req search.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	jobs, err := search.BuildJobs(&req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(jobs) == 0 {
		jsonError(w, "no search jobs generated", http.StatusBadRequest)
		return
	}

	log.Printf("search: %d jobs | origin=%s destinations=%v mode=%s",
		len(jobs), req.Origin, req.Destinations, req.Mode)

	apiKey := os.Getenv("SERPAPI_KEY")
	if apiKey == "" {
		errorMsg := "SERPAPI_KEY environment variable is not set"
		log.Println("error:", errorMsg)
		jsonError(w, errorMsg, http.StatusInternalServerError)
		return
	}
	svc := search.NewService(apiKey)
	results := svc.Run(jobs)

	log.Printf("search done: %d/%d successful", results.Successful, results.TotalJobs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleRegions returns the built-in region→airports mapping.
func handleRegions(w http.ResponseWriter, r *http.Request) {
	regions := map[string][]string{
		"Europe":          {"CDG", "LHR", "FCO", "MAD", "AMS", "FRA", "BCN", "LIS", "ATH"},
		"Balkans":         {"DBV", "SPU", "ZAG", "BEG", "SKP", "TGD", "TIA", "SOF"},
		"SE Asia":         {"BKK", "SIN", "KUL", "HAN", "SGN", "DPS"},
		"North America":   {"JFK", "MIA", "ORD", "LAX", "YYZ"},
		"Central America": {"CUN", "LIR", "PTY", "SJO"},
		"Africa":          {"CPT", "JNB", "NBO", "CMN"},
		"Middle East":     {"DXB", "DOH", "AUH", "IST"},
		"Japan & Korea":   {"NRT", "HND", "KIX", "ICN"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(regions)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
