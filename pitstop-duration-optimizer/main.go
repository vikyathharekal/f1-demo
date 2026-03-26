package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type PitstopAnalysis struct {
	Team            string  `json:"team"`
	AvgDuration     float64 `json:"avg_duration_seconds"`
	OptimalDuration float64 `json:"optimal_duration_seconds"`
	Improvement     float64 `json:"improvement_percent"`
}

var pitstopData = []PitstopAnalysis{
	{Team: "Red Bull", AvgDuration: 2.4, OptimalDuration: 1.8, Improvement: 25.0},
	{Team: "Mercedes", AvgDuration: 2.6, OptimalDuration: 2.0, Improvement: 23.1},
	{Team: "Ferrari", AvgDuration: 2.8, OptimalDuration: 2.1, Improvement: 25.0},
	{Team: "McLaren", AvgDuration: 2.5, OptimalDuration: 1.9, Improvement: 24.0},
}

func handlePitstops(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	team := r.URL.Query().Get("team")
	if team != "" {
		for _, p := range pitstopData {
			if p.Team == team {
				json.NewEncoder(w).Encode(p)
				return
			}
		}
		http.Error(w, `{"error": "team not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(pitstopData)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "pitstop-duration-optimizer",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func main() {
	http.HandleFunc("/api/pitstops", handlePitstops)
	http.HandleFunc("/health", handleHealth)

	log.Println("pitstop-duration-optimizer starting on :7001")
	log.Fatal(http.ListenAndServe(":7001", nil))
}
