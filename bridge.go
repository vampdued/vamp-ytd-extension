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
	URL     string `json:"url"`
	Mode    string `json:"mode"`
	Codec   string `json:"codec"`
	Cookies bool   `json:"cookies"`
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

	fmt.Printf("Received Download request (URL: %s, Mode: %s, Codec: %s, Cookies: %v)\n", p.URL, p.Mode, p.Codec, p.Cookies)

	// Command to open a new visible terminal and run your script
	ytdPath := "ytd"
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		resolvedPath := filepath.Join(execDir, "ytd")
		if _, err := os.Stat(resolvedPath); err == nil {
			ytdPath = resolvedPath
		}
	}

	var ytdArgs []string
	if p.Cookies {
		ytdArgs = append(ytdArgs, "-c")
	}

	var quickOpt string
	switch p.Mode {
	case "quick-max":
		if p.Codec != "" && p.Codec != "auto" {
			quickOpt = p.Codec
		}
	case "quick-1080p":
		if p.Codec != "" && p.Codec != "auto" {
			quickOpt = "1080p," + p.Codec
		} else {
			quickOpt = "1080p"
		}
	case "quick-4k":
		if p.Codec != "" && p.Codec != "auto" {
			quickOpt = "4k," + p.Codec
		} else {
			quickOpt = "4k"
		}
	}

	if p.Mode != "interactive" && p.Mode != "" {
		if quickOpt != "" {
			ytdArgs = append(ytdArgs, "-q", quickOpt)
		} else {
			ytdArgs = append(ytdArgs, "-q")
		}
	}

	ytdArgs = append(ytdArgs, p.URL)

	// Assemble final terminal launch command
	cmdArgs := append([]string{"-e", ytdPath}, ytdArgs...)
	fmt.Printf("Launching terminal: konsole %v\n", cmdArgs)
	cmd := exec.Command("konsole", cmdArgs...)
	
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting script inside terminal:", err)
	}

	w.WriteHeader(http.StatusOK)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("VampYTD Bridge is running correctly! You can close this page."))
}

func installService() {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error resolving executable path: %v\n", err)
		return
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		fmt.Printf("Error resolving absolute path: %v\n", err)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error resolving user home directory: %v\n", err)
		return
	}

	systemdDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		fmt.Printf("Error creating systemd directories: %v\n", err)
		return
	}

	servicePath := filepath.Join(systemdDir, "vampytd-bridge.service")
	serviceContent := fmt.Sprintf(`[Unit]
Description=VampYTD Bridge Server
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
`, execPath)

	err = os.WriteFile(servicePath, []byte(serviceContent), 0644)
	if err != nil {
		fmt.Printf("Error writing systemd service file: %v\n", err)
		return
	}
	fmt.Printf("Successfully auto-generated service file at: %s\n", servicePath)

	fmt.Println("Registering and starting service with systemctl --user...")

	// 1. systemctl --user daemon-reload
	cmdReload := exec.Command("systemctl", "--user", "daemon-reload")
	if err := cmdReload.Run(); err != nil {
		fmt.Printf("Error reloading systemd user daemon: %v\n", err)
		return
	}

	// 2. systemctl --user enable vampytd-bridge.service
	cmdEnable := exec.Command("systemctl", "--user", "enable", "vampytd-bridge.service")
	if err := cmdEnable.Run(); err != nil {
		fmt.Printf("Error enabling vampytd-bridge service: %v\n", err)
		return
	}

	// 3. systemctl --user restart vampytd-bridge.service
	cmdRestart := exec.Command("systemctl", "--user", "restart", "vampytd-bridge.service")
	if err := cmdRestart.Run(); err != nil {
		fmt.Printf("Error restarting/starting vampytd-bridge service: %v\n", err)
		return
	}

	fmt.Println("VampYTD Bridge user service is now successfully active, enabled and running on login!")
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--install" || os.Args[1] == "-i") {
		installService()
		return
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/download", handleDownload)
	fmt.Println("VampYTD Bridge listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server failed:", err)
	}
}
