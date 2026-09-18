package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	baseURL := "http://localhost:8001/tournaments/sleman-open-archery-championship-2026"
	endpoints := []struct {
		filename string
		path     string
	}{
		{"test_scoresheet_qual.pdf", "/qualification/scoresheet"},
		{"test_scoresheet_team.pdf", "/elimination/scoresheet-team?type=team"},
		{"test_scoresheet_mixed.pdf", "/elimination/scoresheet-team?type=mixed"},
		{"test_c75a_brackets.pdf", "/elimination/brackets/printout"},
		{"test_statistics_classes_c03.pdf", "/participants/statistics-classes"},
		{"test_statistics_clubs_c04.pdf", "/participants/statistics-clubs"},
		{"test_c76a_final_ranking_indiv.pdf", "/results/final-ranking/printout?type=individual"},
		{"test_c76b_final_ranking_team.pdf", "/results/final-ranking/printout?type=team"},
		{"test_c73a.pdf", "/qualification/results/printout"},
		{"test_c73c.pdf", "/qualification/results-team/printout"},
		{"test_c93.pdf", "/results/medallists/printout"},
		{"test_c95.pdf", "/results/medals/printout"},
		{"test_c30.pdf", "/entries/by-club/printout"},
		{"test_c32c.pdf", "/qualification/start-list/printout"},
		{"test_c32a.pdf", "/participants/printout?type=alphabetical"},
		{"test_c51a.pdf", "/elimination/start-list/printout"},
		{"test_labels.pdf", "/targets/labels/printout"},
		{"test_c08.pdf", "/schedule/printout"},
		{"test_c58.pdf", "/elimination/schedule/printout"},
	}

	successCount := 0
	for _, ep := range endpoints {
		url := baseURL + ep.path
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("[FAIL] %s (%s): %v\n", ep.filename, url, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("[FAIL] %s (%s): HTTP %d\n", ep.filename, url, resp.StatusCode)
			continue
		}

		out, err := os.Create(ep.filename)
		if err != nil {
			fmt.Printf("[FAIL] %s: could not create file: %v\n", ep.filename, err)
			continue
		}
		n, err := io.Copy(out, resp.Body)
		out.Close()

		if err != nil {
			fmt.Printf("[FAIL] %s: download copy error: %v\n", ep.filename, err)
			continue
		}

		fmt.Printf("[SUCCESS] %s: %d bytes (HTTP 200 OK)\n", ep.filename, n)
		successCount++
	}

	fmt.Printf("\n=== VERIFICATION SUMMARY: %d/%d ENDPOINTS PASSED ===\n", successCount, len(endpoints))
}
