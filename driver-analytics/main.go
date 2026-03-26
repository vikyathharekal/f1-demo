package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var pitstopServiceURL = "http://localhost:7001"

func init() {
	if u := os.Getenv("PITSTOP_SERVICE_URL"); u != "" {
		pitstopServiceURL = u
	}
}

type Driver struct {
	Name       string  `json:"name"`
	Team       string  `json:"team"`
	Points     int     `json:"points"`
	Wins       int     `json:"wins"`
	FastestLap float64 `json:"fastest_lap_seconds"`
}

type DriverWithPitstop struct {
	Driver       Driver      `json:"driver"`
	PitstopStats interface{} `json:"pitstop_stats,omitempty"`
}

var drivers = []Driver{
	{Name: "Max Verstappen", Team: "Red Bull", Points: 575, Wins: 19, FastestLap: 72.109},
	{Name: "Lewis Hamilton", Team: "Mercedes", Points: 234, Wins: 2, FastestLap: 73.421},
	{Name: "Charles Leclerc", Team: "Ferrari", Points: 356, Wins: 5, FastestLap: 72.876},
	{Name: "Lando Norris", Team: "McLaren", Points: 374, Wins: 4, FastestLap: 72.632},
}

var (
	cachedDriverData []DriverWithPitstop
	cacheMu          sync.RWMutex
)

func fetchPitstopData(team string) (interface{}, error) {
	resp, err := http.Get(pitstopServiceURL + "/api/pitstops?team=" + url.QueryEscape(team))
	if err != nil {
		return nil, fmt.Errorf("failed to reach pitstop-duration-optimizer: %w", err)
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

func pollPitstopService() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	refresh := func() {
		var results []DriverWithPitstop
		for _, d := range drivers {
			dp := DriverWithPitstop{Driver: d}
			stats, err := fetchPitstopData(d.Team)
			if err != nil {
				log.Printf("[poll] could not fetch pitstop data for %s: %v", d.Team, err)
			} else {
				dp.PitstopStats = stats
			}
			results = append(results, dp)
		}
		cacheMu.Lock()
		cachedDriverData = results
		cacheMu.Unlock()
		log.Println("[poll] refreshed driver data from pitstop-duration-optimizer")
	}

	refresh()
	for range ticker.C {
		refresh()
	}
}

func handleDrivers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cacheMu.RLock()
	data := cachedDriverData
	cacheMu.RUnlock()
	json.NewEncoder(w).Encode(data)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "driver-analytics",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func main() {
	go pollPitstopService()

	http.HandleFunc("/api/drivers", handleDrivers)
	http.HandleFunc("/health", handleHealth)

	log.Println("driver-analytics starting on :7002")
	log.Fatal(http.ListenAndServe(":7002", nil))
}
