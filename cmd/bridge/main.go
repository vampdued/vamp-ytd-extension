package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

type Payload struct {
	Action  string `json:"action,omitempty"`
	URL     string `json:"url"`
	Mode    string `json:"mode"`
	Codec   string `json:"codec"`
	Cookies bool   `json:"cookies"`
}

const maxRequestBody = 64 * 1024

const (
	nativeHostName  = "com.vampytd.bridge"
	extensionOrigin = "chrome-extension://jjacbochmpbgpfpbfclmileocddkncgd/"
)

var diagnosticWriter io.Writer = os.Stdout
var launchDownloader = launchTerminal

type NativeResponse struct {
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	Platform    string `json:"platform,omitempty"`
	DownloadDir string `json:"downloadDir,omitempty"`
	Downloader  bool   `json:"downloader"`
	YTDLP       bool   `json:"ytDlp"`
	FFmpeg      bool   `json:"ffmpeg"`
	Node        bool   `json:"node"`
	FZF         bool   `json:"fzf"`
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func diagnosticResponse() NativeResponse {
	home, _ := os.UserHomeDir()
	downloadDir := ""
	if home != "" {
		downloadDir = filepath.Join(home, "Downloads", "VampYTD")
	}

	downloaderAvailable := commandAvailable("ytd")
	if executable, err := os.Executable(); err == nil {
		binaryName := "ytd"
		if runtime.GOOS == "windows" {
			binaryName = "ytd.exe"
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(executable), binaryName)); err == nil {
			downloaderAvailable = true
		}
	}

	return NativeResponse{
		OK:          true,
		Platform:    runtime.GOOS,
		DownloadDir: downloadDir,
		Downloader:  downloaderAvailable,
		YTDLP:       commandAvailable("yt-dlp"),
		FFmpeg:      commandAvailable("ffmpeg"),
		Node:        commandAvailable("node"),
		FZF:         commandAvailable("fzf"),
	}
}

type NativeHostManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

func allowedOrigin(origin string) bool {
	if configured := strings.TrimSpace(os.Getenv("VAMPYTD_ALLOWED_ORIGIN")); configured != "" {
		return origin == configured
	}
	return strings.HasPrefix(origin, "chrome-extension://") ||
		strings.HasPrefix(origin, "moz-extension://") ||
		strings.HasPrefix(origin, "extension://")
}

func applyCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if !allowedOrigin(origin) {
		http.Error(w, "Origin is not allowed", http.StatusForbidden)
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Vary", "Origin")
	return true
}

func validatePayload(p Payload) error {
	if p.Action == "ping" {
		return nil
	}
	if p.Action != "" && p.Action != "download" {
		return fmt.Errorf("unsupported action")
	}
	u, err := url.ParseRequestURI(p.URL)
	if err != nil || u.Scheme != "https" {
		return fmt.Errorf("a valid HTTPS video URL is required")
	}
	host := strings.ToLower(u.Hostname())
	allowedHost := host == "youtube.com" || strings.HasSuffix(host, ".youtube.com") ||
		host == "youtu.be" || host == "hotstar.com" || strings.HasSuffix(host, ".hotstar.com") ||
		host == "jiohotstar.com" || strings.HasSuffix(host, ".jiohotstar.com")
	if !allowedHost {
		return fmt.Errorf("unsupported video host")
	}
	validModes := map[string]bool{"": true, "interactive": true, "quick-max": true, "quick-1080p": true, "quick-4k": true}
	validCodecs := map[string]bool{"": true, "auto": true, "av1": true, "vp9": true, "hevc": true, "h264": true}
	if !validModes[p.Mode] || !validCodecs[p.Codec] {
		return fmt.Errorf("unsupported mode or codec")
	}
	return nil
}

func processPayload(p Payload) error {
	if err := validatePayload(p); err != nil {
		return err
	}
	if p.Action == "ping" {
		return nil
	}

	ytdPath := "ytd"
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		binaryName := "ytd"
		if runtime.GOOS == "windows" {
			binaryName = "ytd.exe"
		}
		resolvedPath := filepath.Join(execDir, binaryName)
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
	return launchDownloader(ytdPath, ytdArgs)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if !applyCORS(w, r) {
		return
	}

	// Handle preflight OPTIONS request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody))
	decoder.DisallowUnknownFields()
	var p Payload
	if err := decoder.Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(diagnosticWriter, "Received download request (URL: %s, Mode: %s, Codec: %s, Cookies: %v)\n", p.URL, p.Mode, p.Codec, p.Cookies)
	if err := processPayload(p); err != nil {
		fmt.Println("Error starting script inside terminal:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func powershellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func windowsPathDir(value string) string {
	normalized := strings.ReplaceAll(value, `\`, "/")
	return strings.ReplaceAll(path.Dir(normalized), "/", `\`)
}

func buildWindowsPowerShellScript(ytdPath string, ytdArgs []string) string {
	quotedArgs := make([]string, 0, len(ytdArgs))
	for _, arg := range ytdArgs {
		quotedArgs = append(quotedArgs, powershellSingleQuote(arg))
	}
	return `$arguments = @(` + strings.Join(quotedArgs, ", ") + `); ` +
		`Start-Process -FilePath ` + powershellSingleQuote(ytdPath) +
		` -ArgumentList $arguments -WorkingDirectory ` + powershellSingleQuote(windowsPathDir(ytdPath))
}

func encodePowerShellCommand(script string) string {
	codeUnits := utf16.Encode([]rune(script))
	data := make([]byte, len(codeUnits)*2)
	for i, codeUnit := range codeUnits {
		binary.LittleEndian.PutUint16(data[i*2:], codeUnit)
	}
	return base64.StdEncoding.EncodeToString(data)
}

// displayEnv returns the display-related environment variables needed for GUI
// apps. When the bridge runs as a systemd service its environment is minimal
// (no DISPLAY / WAYLAND_DISPLAY), so we must forward these explicitly to any
// child process that opens a window, otherwise Qt calls abort() at startup.
func displayEnv() []string {
	keys := []string{
		"DISPLAY",
		"WAYLAND_DISPLAY",
		"XDG_RUNTIME_DIR",
		"DBUS_SESSION_BUS_ADDRESS",
		"XDG_SESSION_TYPE",
		"XDG_CURRENT_DESKTOP",
	}
	var env []string
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func launchTerminal(ytdPath string, ytdArgs []string) error {
	if runtime.GOOS == "windows" {
		if terminalPath, err := exec.LookPath("wt.exe"); err == nil {
			args := []string{"new-tab", "--startingDirectory", filepath.Dir(ytdPath), "--title", "VampYTD Downloader", ytdPath}
			args = append(args, ytdArgs...)
			return exec.Command(terminalPath, args...).Start()
		}

		// Start-Process calls CreateProcess directly and therefore does not expose
		// URLs to cmd.exe metacharacter parsing. EncodedCommand also avoids another
		// quoting layer when the installation path contains spaces.
		script := buildWindowsPowerShellScript(ytdPath, ytdArgs)
		return exec.Command(
			"powershell.exe",
			"-NoProfile",
			"-NonInteractive",
			"-WindowStyle", "Hidden",
			"-EncodedCommand", encodePowerShellCommand(script),
		).Start()
	}
	if runtime.GOOS == "darwin" {
		parts := []string{posixShellQuote(ytdPath)}
		for _, arg := range ytdArgs {
			parts = append(parts, posixShellQuote(arg))
		}
		commandLine := strings.Join(parts, " ")
		script := `tell application "Terminal" to do script "` + appleScriptEscape(commandLine) + `"`
		return exec.Command("osascript", "-e", script).Start()
	}

	// On Linux/macOS, scan for terminal emulators
	terminals := []string{"konsole", "gnome-terminal", "xfce4-terminal", "alacritty", "kitty", "xterm"}
	var foundTerminal string
	for _, term := range terminals {
		if _, err := exec.LookPath(term); err == nil {
			foundTerminal = term
			break
		}
	}

	if foundTerminal == "" {
		// Fallback: spawn the raw process in the background without terminal wrapper
		fmt.Fprintln(diagnosticWriter, "No terminal emulator found. Spawning raw background process...")
		args := append([]string{}, ytdArgs...)
		cmd := exec.Command(ytdPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Start()
	}

	var cmdArgs []string
	switch foundTerminal {
	case "gnome-terminal":
		// gnome-terminal -- ytd args...
		cmdArgs = append([]string{"--", ytdPath}, ytdArgs...)
	case "kitty":
		// kitty ytd args...
		cmdArgs = append([]string{ytdPath}, ytdArgs...)
	default:
		// konsole, xfce4-terminal, alacritty, xterm all support -e <cmd> [args]
		cmdArgs = append([]string{"-e", ytdPath}, ytdArgs...)
	}

	fmt.Fprintf(diagnosticWriter, "Launching terminal: %s %v\n", foundTerminal, cmdArgs)
	cmd := exec.Command(foundTerminal, cmdArgs...)
	// Inherit the current environment and overlay display vars. This ensures
	// GUI terminals can connect to the display even when the bridge was started
	// by systemd (which strips session-specific env vars like DISPLAY).
	cmd.Env = append(os.Environ(), displayEnv()...)
	return cmd.Start()
}

func posixShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func appleScriptEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func readNativeMessage(reader io.Reader) (Payload, error) {
	var length uint32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return Payload{}, err
	}
	if length == 0 || length > maxRequestBody {
		return Payload{}, fmt.Errorf("native message length %d is invalid", length)
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return Payload{}, err
	}
	var payload Payload
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func writeNativeMessage(writer io.Writer, response NativeResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(len(body))); err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}

func runNativeHost(reader io.Reader, writer io.Writer) error {
	bufferedReader := bufio.NewReader(reader)
	for {
		payload, err := readNativeMessage(bufferedReader)
		if err == io.EOF {
			return nil
		}
		response := NativeResponse{OK: true}
		if err != nil {
			response = NativeResponse{OK: false, Error: err.Error()}
		} else if err := processPayload(payload); err != nil {
			response = NativeResponse{OK: false, Error: err.Error()}
		} else if payload.Action == "ping" {
			response = diagnosticResponse()
		}
		if err := writeNativeMessage(writer, response); err != nil {
			return err
		}
		if err != nil {
			return nil
		}
	}
}

func isNativeInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return args[0] == "--native-host" || strings.TrimSuffix(args[0], "/") == strings.TrimSuffix(extensionOrigin, "/")
}

func nativeManifestPaths() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	name := nativeHostName + ".json"
	switch runtime.GOOS {
	case "windows":
		executable, err := os.Executable()
		if err != nil {
			return nil, err
		}
		return []string{filepath.Join(filepath.Dir(executable), name)}, nil
	case "darwin":
		base := filepath.Join(home, "Library", "Application Support")
		return []string{
			filepath.Join(base, "Google", "Chrome", "NativeMessagingHosts", name),
			filepath.Join(base, "Google", "ChromeForTesting", "NativeMessagingHosts", name),
			filepath.Join(base, "Chromium", "NativeMessagingHosts", name),
			filepath.Join(base, "Microsoft Edge", "NativeMessagingHosts", name),
			filepath.Join(base, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts", name),
			filepath.Join(base, "Vivaldi", "NativeMessagingHosts", name),
		}, nil
	default:
		configRoot, err := os.UserConfigDir()
		if err != nil {
			configRoot = filepath.Join(home, ".config")
		}
		return []string{
			filepath.Join(configRoot, "google-chrome", "NativeMessagingHosts", name),
			filepath.Join(configRoot, "google-chrome-for-testing", "NativeMessagingHosts", name),
			filepath.Join(configRoot, "chromium", "NativeMessagingHosts", name),
			filepath.Join(configRoot, "microsoft-edge", "NativeMessagingHosts", name),
			filepath.Join(configRoot, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts", name),
			filepath.Join(configRoot, "vivaldi", "NativeMessagingHosts", name),
		}, nil
	}
}

func installNativeHost() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return err
	}
	manifest := NativeHostManifest{
		Name:           nativeHostName,
		Description:    "VampYTD browser integration",
		Path:           executable,
		Type:           "stdio",
		AllowedOrigins: []string{extensionOrigin},
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	paths, err := nativeManifestPaths()
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, body, 0644); err != nil {
			return err
		}
	}
	if runtime.GOOS == "windows" {
		registries := []string{
			`HKCU\Software\Google\Chrome\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\Microsoft\Edge\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\Chromium\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\Vivaldi\NativeMessagingHosts\` + nativeHostName,
		}
		for _, key := range registries {
			cmd := exec.Command("reg.exe", "add", key, "/ve", "/t", "REG_SZ", "/d", paths[0], "/f")
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("registering %s: %w: %s", key, err, strings.TrimSpace(string(output)))
			}
		}
	}
	fmt.Printf("Registered native messaging host %s for extension %s\n", nativeHostName, extensionOrigin)
	return nil
}

func uninstallNativeHost() error {
	paths, err := nativeManifestPaths()
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if runtime.GOOS == "windows" {
		registries := []string{
			`HKCU\Software\Google\Chrome\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\Microsoft\Edge\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\Chromium\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\` + nativeHostName,
			`HKCU\Software\Vivaldi\NativeMessagingHosts\` + nativeHostName,
		}
		for _, key := range registries {
			_ = exec.Command("reg.exe", "delete", key, "/f").Run()
		}
	}
	fmt.Printf("Unregistered native messaging host %s\n", nativeHostName)
	return nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "" && !applyCORS(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("VampYTD Bridge is running correctly! You can close this page."))
}

func handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if !applyCORS(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(diagnosticResponse())
}

func installService() {
	if runtime.GOOS != "linux" {
		fmt.Println("Automatic bridge service installation is currently supported on Linux only.")
		return
	}

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

	// Capture current display session vars at install time so the service
	// can spawn GUI apps (e.g. konsole) that require a display connection.
	// Without these, Qt/X11/Wayland will abort with SIGABRT at init_platform().
	display := os.Getenv("DISPLAY")
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	dbusAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")

	envLine := ""
	if display != "" {
		envLine += fmt.Sprintf("Environment=DISPLAY=%s\n", display)
	}
	if waylandDisplay != "" {
		envLine += fmt.Sprintf("Environment=WAYLAND_DISPLAY=%s\n", waylandDisplay)
	}
	if xdgRuntime != "" {
		envLine += fmt.Sprintf("Environment=XDG_RUNTIME_DIR=%s\n", xdgRuntime)
	}
	if dbusAddr != "" {
		envLine += fmt.Sprintf("Environment=DBUS_SESSION_BUS_ADDRESS=%s\n", dbusAddr)
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=VampYTD Bridge Server
After=network.target

[Service]
Type=simple
ExecStart=%s
%sRestart=always
RestartSec=3

[Install]
WantedBy=default.target
`, execPath, envLine)

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
	if isNativeInvocation(os.Args[1:]) {
		diagnosticWriter = os.Stderr
		if err := runNativeHost(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "Native host failed:", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && (os.Args[1] == "--install" || os.Args[1] == "-i") {
		installService()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--install-native" {
		if err := installNativeHost(); err != nil {
			fmt.Fprintln(os.Stderr, "Native host installation failed:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--uninstall-native" {
		if err := uninstallNativeHost(); err != nil {
			fmt.Fprintln(os.Stderr, "Native host removal failed:", err)
			os.Exit(1)
		}
		return
	}

	port := 8080
	if value := os.Getenv("VAMPYTD_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			fmt.Println("Invalid VAMPYTD_PORT; expected a number from 1 to 65535")
			return
		}
		port = parsed
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/diagnostics", handleDiagnostics)
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/download", handleDownload)
	address := fmt.Sprintf("127.0.0.1:%d", port)
	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	fmt.Printf("VampYTD Bridge listening on http://%s\n", address)
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Server failed:", err)
	}
}
