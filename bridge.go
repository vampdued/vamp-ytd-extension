package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type Payload struct {
	URL string `json:"url"`
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers so Chrome doesn't block the request
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var p Payload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("Received URL:", p.URL)

	// Command to open a new visible terminal and run your script
	ytdPath := "ytd"
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		resolvedPath := filepath.Join(execDir, "ytd")
		if _, err := os.Stat(resolvedPath); err == nil {
			ytdPath = resolvedPath
		}
	}

	cmd := exec.Command("konsole", "-e", ytdPath, p.URL)
	
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting script:", err)
	}

	w.WriteHeader(http.StatusOK)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("VampYTD Bridge is running correctly! You can close this page."))
}

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/download", handleDownload)
	fmt.Println("VampYTD Bridge listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server failed:", err)
	}
}
