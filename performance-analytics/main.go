package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	pitstopServiceURL = "http://localhost:7001"
	driverServiceURL  = "http://localhost:7002"
)

func init() {
	if u := os.Getenv("PITSTOP_SERVICE_URL"); u != "" {
		pitstopServiceURL = u
	}
	if u := os.Getenv("DRIVER_SERVICE_URL"); u != "" {
		driverServiceURL = u
	}
}

type PerformanceReport struct {
	GeneratedAt string      `json:"generated_at"`
	Drivers     interface{} `json:"drivers"`
	Pitstops    interface{} `json:"pitstops"`
}

var (
	cachedReport *PerformanceReport
	cacheMu      sync.RWMutex
)

func fetchJSON(url string) (interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func pollUpstreamServices() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	refresh := func() {
		drivers, driverErr := fetchJSON(driverServiceURL + "/api/drivers")
		if driverErr != nil {
			log.Printf("[poll] could not fetch driver data: %v", driverErr)
		}

		pitstops, pitstopErr := fetchJSON(pitstopServiceURL + "/api/pitstops")
		if pitstopErr != nil {
			log.Printf("[poll] could not fetch pitstop data: %v", pitstopErr)
		}

		report := &PerformanceReport{
			GeneratedAt: time.Now().Format(time.RFC3339),
			Drivers:     drivers,
			Pitstops:    pitstops,
		}

		cacheMu.Lock()
		cachedReport = report
		cacheMu.Unlock()
		log.Println("[poll] refreshed performance report from upstream services")
	}

	refresh()
	for range ticker.C {
		refresh()
	}
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cacheMu.RLock()
	report := cachedReport
	cacheMu.RUnlock()

	if report == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "data not yet available"})
		return
	}

	json.NewEncoder(w).Encode(report)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "performance-analytics",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func main() {
	go pollUpstreamServices()

	http.HandleFunc("/api/report", handleReport)
	http.HandleFunc("/health", handleHealth)

	log.Println("performance-analytics starting on :7003")
	log.Fatal(http.ListenAndServe(":7003", nil))
}
