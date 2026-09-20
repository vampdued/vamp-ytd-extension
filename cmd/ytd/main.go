package main

// VampYTD — Go port of ytd.ps1
// Feature Complete implementation according to YTD Core Feature Specification

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// ==============================================================================
// CONFIG
// ==============================================================================

var (
	cookiesYT   string
	cookiesJHS  string
	downloadDir string
	isSpawned   bool
)

const (
	ytdlp  = "yt-dlp"
	ffmpeg = "ffmpeg"
)

var (
	clientArgs = []string{"--js-runtime", "node"}
	template   = `%(uploader)s - %(title)s [%(id)s] [%(height)sp] [%(vcodec)s] [%(format_id)s].%(ext)s`
)

// ==============================================================================
// ANSI COLOUR HELPERS
// ==============================================================================

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorGray   = "\033[90m"
	colorDkCyan = "\033[96m"
)

func cyan(s string) string   { return colorCyan + s + colorReset }
func green(s string) string  { return colorGreen + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }
func red(s string) string    { return colorRed + s + colorReset }
func gray(s string) string   { return colorGray + s + colorReset }
func dkCyan(s string) string { return colorDkCyan + s + colorReset }

func stripANSI(str string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(str, "")
}

// ==============================================================================
// HELPERS
// ==============================================================================

var (
	modkernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procExpandEnvironmentStringsW = modkernel32.NewProc("ExpandEnvironmentStringsW")
)

func expandWinEnv(input string) string {
	if input == "" {
		return ""
	}
	ptr, err := syscall.UTF16PtrFromString(input)
	if err != nil {
		return input
	}
	buf := make([]uint16, 32768)
	r1, _, _ := procExpandEnvironmentStringsW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r1 == 0 {
		return input
	}
	return syscall.UTF16ToString(buf)
}

func getRegistryEnv(key syscall.Handle, subkey, valueName string) string {
	var hKey syscall.Handle
	subKeyPtr, _ := syscall.UTF16PtrFromString(subkey)
	if err := syscall.RegOpenKeyEx(key, subKeyPtr, 0, syscall.KEY_READ, &hKey); err != nil {
		return ""
	}
	defer syscall.RegCloseKey(hKey)

	valPtr, _ := syscall.UTF16PtrFromString(valueName)
	var bufSize uint32
	var valType uint32
	if err := syscall.RegQueryValueEx(hKey, valPtr, nil, &valType, nil, &bufSize); err != nil {
		return ""
	}

	buf := make([]uint16, bufSize/2+1)
	if err := syscall.RegQueryValueEx(hKey, valPtr, nil, &valType, (*byte)(unsafe.Pointer(&buf[0])), &bufSize); err != nil {
		return ""
	}

	return syscall.UTF16ToString(buf)
}

func refreshPATH() {
	if runtime.GOOS != "windows" {
		return
	}
	userPath := getRegistryEnv(syscall.HKEY_CURRENT_USER, "Environment", "Path")
	machinePath := getRegistryEnv(syscall.HKEY_LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, "Path")

	rawPath := userPath
	if machinePath != "" {
		if rawPath != "" {
			rawPath = rawPath + ";" + machinePath
		} else {
			rawPath = machinePath
		}
	}
	if current := os.Getenv("PATH"); current != "" {
		rawPath = rawPath + ";" + current
	}

	expanded := expandWinEnv(rawPath)

	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	extraPaths := []string{}
	if localAppData != "" {
		extraPaths = append(extraPaths,
			filepath.Join(localAppData, "Microsoft", "WinGet", "Links"),
		)
		packagesDir := filepath.Join(localAppData, "Microsoft", "WinGet", "Packages")
		if entries, err := os.ReadDir(packagesDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					pkgPath := filepath.Join(packagesDir, entry.Name())
					extraPaths = append(extraPaths, pkgPath)
					if subEntries, err := os.ReadDir(pkgPath); err == nil {
						for _, sub := range subEntries {
							if sub.IsDir() {
								if strings.EqualFold(sub.Name(), "bin") {
									extraPaths = append(extraPaths, filepath.Join(pkgPath, sub.Name()))
								} else {
									subPath := filepath.Join(pkgPath, sub.Name())
									if subSubEntries, err := os.ReadDir(subPath); err == nil {
										for _, subSub := range subSubEntries {
											if subSub.IsDir() && strings.EqualFold(subSub.Name(), "bin") {
												extraPaths = append(extraPaths, filepath.Join(subPath, subSub.Name()))
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if programFiles != "" {
		extraPaths = append(extraPaths, filepath.Join(programFiles, "nodejs"))
	}
	if programFilesX86 != "" {
		extraPaths = append(extraPaths, filepath.Join(programFilesX86, "nodejs"))
	}

	allEntries := append(strings.Split(expanded, ";"), extraPaths...)
	seen := make(map[string]bool)
	var finalEntries []string
	for _, p := range allEntries {
		p = strings.TrimSpace(p)
		if p == "" || seen[strings.ToLower(p)] {
			continue
		}
		seen[strings.ToLower(p)] = true
		finalEntries = append(finalEntries, p)
	}

	os.Setenv("PATH", strings.Join(finalEntries, ";"))
}

func commandExists(name string) bool {
	refreshPATH()
	_, err := exec.LookPath(name)
	return err == nil
}

func encodePowerShellCommand(script string) string {
	codeUnits := utf16.Encode([]rune(script))
	data := make([]byte, len(codeUnits)*2)
	for i, codeUnit := range codeUnits {
		binary.LittleEndian.PutUint16(data[i*2:], codeUnit)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func showToast(title, message string) {
	if runtime.GOOS != "windows" {
		return
	}
	cleanTitle := strings.ReplaceAll(title, "'", "''")
	cleanMsg := strings.ReplaceAll(message, "'", "''")

	script := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$nodes = $template.GetElementsByTagName('text')
$nodes.Item(0).AppendChild($template.CreateTextNode('%s')) | Out-Null
$nodes.Item(1).AppendChild($template.CreateTextNode('%s')) | Out-Null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('VampYTD').Show($toast)`, cleanTitle, cleanMsg)

	_ = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-EncodedCommand", encodePowerShellCommand(script)).Start()
}

func runUpdate() {
	refreshPATH()
	fmt.Println(cyan(">> Checking for yt-dlp updates..."))

	// 1. Try yt-dlp -U
	cmd := exec.Command(ytdlp, "-U")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err == nil {
		fmt.Println(green(">> yt-dlp updated successfully."))
		return
	}

	// 2. If self-update failed, try winget on Windows
	if runtime.GOOS == "windows" && commandExists("winget") {
		fmt.Println(yellow(">> Self-update failed; attempting update via Windows Package Manager (winget)..."))
		wingetCmd := exec.Command("winget", "upgrade", "--exact", "--id", "yt-dlp.yt-dlp", "--accept-source-agreements", "--accept-package-agreements")
		wingetCmd.Stdout = os.Stdout
		wingetCmd.Stderr = os.Stderr
		if err := wingetCmd.Run(); err == nil {
			fmt.Println(green(">> yt-dlp updated successfully via winget."))
			return
		}
	}

	die("Failed to update yt-dlp. You can update manually with: winget upgrade yt-dlp.yt-dlp")
}

func checkYTDLPAge(configRoot string) {
	checkFile := filepath.Join(configRoot, "vampytd", ".last_update_check")
	if info, err := os.Stat(checkFile); err == nil {
		if time.Since(info.ModTime()) < 7*24*time.Hour {
			return
		}
	}

	// Record check timestamp
	_ = os.MkdirAll(filepath.Dir(checkFile), 0755)
	_ = os.WriteFile(checkFile, []byte(time.Now().Format(time.RFC3339)), 0644)

	out, err := exec.Command(ytdlp, "--version").Output()
	if err != nil || len(out) == 0 {
		return
	}
	vStr := strings.TrimSpace(string(out))
	parts := strings.Split(vStr, ".")
	if len(parts) >= 3 {
		dateStr := fmt.Sprintf("%s.%s.%s", parts[0], parts[1], parts[2])
		if vDate, err := time.Parse("2006.01.02", dateStr); err == nil {
			if time.Since(vDate) > 30*24*time.Hour {
				fmt.Println(yellow(fmt.Sprintf(">> [Notice] Your yt-dlp version (%s) is older than 30 days. Run 'ytd -U' to update.", vStr)))
			}
		}
	}
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, red("Error: ")+msg)
	if isSpawned {
		fmt.Fprintln(os.Stderr, yellow("\nPress Enter to close this window..."))
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
	os.Exit(1)
}

func prompt(label string) string {
	fmt.Print(label)
	sc := bufio.NewScanner(os.Stdin)
	sc.Scan()
	return strings.TrimSpace(sc.Text())
}

// ==============================================================================
// CODEC MATCHING DICTIONARY
// ==============================================================================

func matchesCodec(vcodec, reqCodec string) bool {
	v := strings.ToLower(vcodec)
	req := strings.ToLower(reqCodec)

	switch req {
	case "av1", "av01":
		return strings.Contains(v, "av1") || strings.Contains(v, "av01")
	case "vp9", "vp09":
		return strings.Contains(v, "vp9") || strings.Contains(v, "vp09")
	case "hevc", "h265", "h.265":
		return strings.Contains(v, "hevc") || strings.Contains(v, "h265") || strings.Contains(v, "h.265") ||
			strings.Contains(v, "hev1") || strings.Contains(v, "hvc1")
	case "h264", "h.264", "avc", "avc1":
		return strings.Contains(v, "h264") || strings.Contains(v, "h.264") || strings.Contains(v, "avc")
	default:
		// Fallback to fuzzy substring match
		return strings.Contains(v, req)
	}
}

// ==============================================================================
// ARGUMENT PARSING
// ==============================================================================

func parseArgs() (updateMode bool, quickMode bool, quickOptions string, trimMode, useCookies bool, startTime, endTime, url string) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-U" || arg == "--update" || arg == "--upgrade":
			updateMode = true
		case arg == "--spawned":
			isSpawned = true
		case arg == "-q" || arg == "--quick":
			quickMode = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.HasPrefix(args[i+1], "http") {
				quickOptions = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-q="):
			quickMode = true
			quickOptions = arg[3:]
		case arg == "-c" || arg == "--cookies":
			useCookies = true
		case arg == "-t" || arg == "--trim":
			trimMode = true
			if i+1 < len(args) && regexp.MustCompile(`^\d`).MatchString(args[i+1]) {
				i++
				startTime = args[i]
				if i+1 < len(args) && regexp.MustCompile(`^\d`).MatchString(args[i+1]) {
					i++
					endTime = args[i]
				}
			}
		case arg == "-h" || arg == "--help":
			fmt.Println(`VampYTD`)
			fmt.Println(`Usage: VampYTD [FLAGS] "URL"`)
			fmt.Println(`  -q [opts] : Quick Mode (e.g. -q 1080p,av1). Leave empty for fastest default max-quality download.`)
			fmt.Println(`  -c        : Enable Cookies (Auto selects file)`)
			fmt.Println(`  -t        : Trim Mode (e.g. -t 8:20 12:20)`)
			fmt.Println(`  -U        : Update yt-dlp to latest version`)
			os.Exit(0)
		default:
			if strings.HasPrefix(arg, "http") {
				url = arg
			}
		}
	}
	return
}

// ==============================================================================
// COOKIE RESOLUTION
// ==============================================================================

func resolveCookies(url string) []string {
	reHS := regexp.MustCompile(`hotstar|jiohotstar|hs\.`)
	reYT := regexp.MustCompile(`youtube|youtu\.be`)

	var selectedFile string
	switch {
	case reHS.MatchString(url):
		selectedFile = cookiesJHS
		fmt.Println(green(">> [Cookies] Hotstar Mode (cookies-jhs.txt)"))
	case reYT.MatchString(url):
		selectedFile = cookiesYT
		fmt.Println(green(">> [Cookies] YouTube Mode (cookies-yt.txt)"))
	default:
		selectedFile = cookiesYT
		fmt.Println(dkCyan(">> [Cookies] Default Mode (cookies-yt.txt)"))
	}

	if _, err := os.Stat(selectedFile); err == nil {
		return []string{"--cookies", selectedFile}
	}
	fmt.Println(yellow(">> [Warning] Cookie file missing: " + selectedFile))
	return nil
}

// ==============================================================================
// TRIM RESOLUTION
// ==============================================================================

func resolveTrim(startTime, endTime *string) []string {
	if strings.TrimSpace(*startTime) == "" {
		fmt.Println("------------------------------------------------")
		fmt.Println("TRIM MODE ACTIVE (Enter timestamps)")
		*startTime = prompt("Start Time (e.g. 08:20): ")
		*endTime = prompt("End Time   (e.g. 12:20): ")
	}

	if strings.TrimSpace(*startTime) == "" || strings.TrimSpace(*endTime) == "" {
		die("Timestamps cannot be empty.")
	}

	fmt.Println("------------------------------------------------")
	fmt.Printf(">> [TRIM ENABLED] Cutting: %s to %s\n", *startTime, *endTime)
	fmt.Println(">> [MODE] Stream Copy via ffmpeg.")
	fmt.Println("------------------------------------------------")

	return []string{
		"--download-sections", "*" + *startTime + "-" + *endTime,
		"--downloader", "ffmpeg",
	}
}

// ==============================================================================
// METADATA DEFINITIONS (JSON)
// ==============================================================================

type VideoMetadata struct {
	Title          string      `json:"title"`
	Uploader       string      `json:"uploader"`
	ID             string      `json:"id"`
	DurationString string      `json:"duration_string"`
	Thumbnail      string      `json:"thumbnail"`
	Formats        []FormatRaw `json:"formats"`
}

type FormatRaw struct {
	FormatID       string   `json:"format_id"`
	Ext            string   `json:"ext"`
	Resolution     *string  `json:"resolution"`
	Height         *int     `json:"height"`
	Width          *int     `json:"width"`
	FPS            *float64 `json:"fps"`
	VCodec         string   `json:"vcodec"`
	ACodec         string   `json:"acodec"`
	VBR            *float64 `json:"vbr"`
	ABR            *float64 `json:"abr"`
	TBR            *float64 `json:"tbr"`
	ASR            *float64 `json:"asr"`
	Language       *string  `json:"language"`
	Filesize       *int64   `json:"filesize"`
	FilesizeApprox *int64   `json:"filesize_approx"`
	DynamicRange   *string  `json:"dynamic_range"`
	FormatNote     *string  `json:"format_note"`
}

type Format struct {
	Raw             FormatRaw
	IsVideoOnly     bool
	IsAudioOnly     bool
	IsMuxed         bool
	EffectiveSize   int64
	ResolutionLabel string
	CodecRank       int
	HDRRank         int
	AudioRank       int
}

// ==============================================================================
// FEATURE 1: yt-dlp JSON PARSER
// ==============================================================================

func fetchMetadata(url string, cookieCmd []string) VideoMetadata {
	args := append(clientArgs, cookieCmd...)
	args = append(args, "-J", "--no-playlist", url)

	fmt.Println(yellow(">> Fetching metadata..."))
	out, err := exec.Command(ytdlp, args...).Output()
	if err != nil {
		var exitCode int
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		die(fmt.Sprintf("Metadata fetch failed (exit %d). Check URL or enable cookies.", exitCode))
	}

	if len(out) == 0 {
		die("yt-dlp returned no data.")
	}

	var meta VideoMetadata
	if err := json.Unmarshal(out, &meta); err != nil {
		os.WriteFile("ytd-error.log", out, 0644)
		die("Could not parse yt-dlp output as JSON. Raw output saved to ytd-error.log")
	}

	if len(meta.Formats) == 0 {
		die("No formats found in metadata.")
	}

	return meta
}

// ==============================================================================
// FEATURE 2: SMART FORMAT SORTING
// ==============================================================================

func getCodecRank(vcodec string) int {
	v := strings.ToLower(vcodec)
	if strings.HasPrefix(v, "av01") {
		return 1
	}
	if strings.HasPrefix(v, "vp9") || strings.HasPrefix(v, "vp09") {
		return 2
	}
	if strings.HasPrefix(v, "h265") || strings.HasPrefix(v, "hevc") || strings.HasPrefix(v, "hev1") || strings.HasPrefix(v, "hvc1") {
		return 3
	}
	if strings.HasPrefix(v, "h264") || strings.HasPrefix(v, "avc1") {
		return 4
	}
	return 99
}

func getAudioRank(acodec string) int {
	a := strings.ToLower(acodec)
	if strings.Contains(a, "opus") {
		return 1
	}
	if strings.Contains(a, "aac") || strings.Contains(a, "mp4a") {
		return 2
	}
	if strings.Contains(a, "mp3") {
		return 3
	}
	return 99
}

func processFormats(meta VideoMetadata) []Format {
	var formats []Format
	for _, raw := range meta.Formats {
		// Filter out storyboards and metadata-only streams
		if raw.VCodec == "none" && raw.ACodec == "none" {
			continue
		}
		note := ""
		if raw.FormatNote != nil {
			note = strings.ToLower(*raw.FormatNote)
		}
		if strings.Contains(note, "storyboard") || raw.VCodec == "mhtml" {
			continue
		}

		f := Format{Raw: raw}

		f.IsVideoOnly = raw.VCodec != "none" && raw.ACodec == "none"
		f.IsAudioOnly = raw.VCodec == "none" && raw.ACodec != "none"
		f.IsMuxed = raw.VCodec != "none" && raw.ACodec != "none"

		if raw.Filesize != nil {
			f.EffectiveSize = *raw.Filesize
		} else if raw.FilesizeApprox != nil {
			f.EffectiveSize = *raw.FilesizeApprox
		} else {
			f.EffectiveSize = 0
		}

		if raw.Height != nil {
			f.ResolutionLabel = fmt.Sprintf("%dp", *raw.Height)
		} else {
			f.ResolutionLabel = "audio"
		}

		f.CodecRank = getCodecRank(raw.VCodec)
		f.AudioRank = getAudioRank(raw.ACodec)

		f.HDRRank = 1
		if raw.DynamicRange != nil && *raw.DynamicRange != "SDR" {
			f.HDRRank = 0
		}

		formats = append(formats, f)
	}

	sort.SliceStable(formats, func(i, j int) bool {
		a, b := formats[i], formats[j]

		// Audio-only at the bottom
		if a.IsAudioOnly != b.IsAudioOnly {
			return !a.IsAudioOnly
		}

		// Both audio-only
		if a.IsAudioOnly && b.IsAudioOnly {
			if a.AudioRank != b.AudioRank {
				return a.AudioRank < b.AudioRank
			}
			return a.EffectiveSize > b.EffectiveSize
		}

		// Video or Muxed
		var aHeight, bHeight int
		if a.Raw.Height != nil {
			aHeight = *a.Raw.Height
		}
		if b.Raw.Height != nil {
			bHeight = *b.Raw.Height
		}

		if aHeight != bHeight {
			return aHeight > bHeight
		}

		// Sort by FPS (60fps > 30fps)
		var aFPS, bFPS float64
		if a.Raw.FPS != nil {
			aFPS = *a.Raw.FPS
		}
		if b.Raw.FPS != nil {
			bFPS = *b.Raw.FPS
		}
		if aFPS != bFPS {
			return aFPS > bFPS
		}

		if a.HDRRank != b.HDRRank {
			return a.HDRRank < b.HDRRank
		}

		if a.CodecRank != b.CodecRank {
			return a.CodecRank < b.CodecRank
		}

		// Prefer higher bitrate / size for better quality
		return a.EffectiveSize > b.EffectiveSize
	})

	return formats
}

// ==============================================================================
// FEATURE 4: FZF MULTI-SELECT FORMAT PICKER
// ==============================================================================

func formatSize(size int64) string {
	if size == 0 {
		return "?"
	}
	fSize := float64(size)
	if size >= 1000000000 {
		return fmt.Sprintf("%.1f GB", fSize/1000000000)
	}
	if size >= 1000000 {
		return fmt.Sprintf("%.1f MB", fSize/1000000)
	}
	return fmt.Sprintf("%.0f KB", fSize/1000)
}

func getFlags(f Format) string {
	var flags []string
	if f.Raw.DynamicRange != nil && *f.Raw.DynamicRange != "SDR" {
		flags = append(flags, *f.Raw.DynamicRange)
	}
	if f.IsVideoOnly {
		flags = append(flags, "video-only")
	}
	if f.IsAudioOnly {
		flags = append(flags, "audio-only")
	}
	if f.Raw.FormatNote != nil && strings.Contains(strings.ToLower(*f.Raw.FormatNote), "premium") {
		flags = append(flags, "premium")
	}
	if f.Raw.Language != nil && *f.Raw.Language != "" {
		flags = append(flags, "lang:"+*f.Raw.Language)
	}
	return strings.Join(flags, " ")
}

func formatLine(f Format) string {
	vcodec := f.Raw.VCodec
	if idx := strings.Index(vcodec, "."); idx != -1 {
		vcodec = vcodec[:idx]
	}
	if vcodec == "none" {
		vcodec = "-"
	}

	acodec := f.Raw.ACodec
	if idx := strings.Index(acodec, "."); idx != -1 {
		acodec = acodec[:idx]
	}
	if acodec == "none" {
		acodec = "-"
	}

	fpsStr := "-"
	if f.Raw.FPS != nil && *f.Raw.FPS > 0 {
		fpsStr = fmt.Sprintf("%.0f", *f.Raw.FPS)
	}

	sizeStr := formatSize(f.EffectiveSize)

	bitrateStr := "-"
	if f.Raw.TBR != nil && *f.Raw.TBR > 0 {
		bitrateStr = fmt.Sprintf("%.0fK", *f.Raw.TBR)
	} else if f.Raw.VBR != nil && *f.Raw.VBR > 0 {
		bitrateStr = fmt.Sprintf("v%.0fK", *f.Raw.VBR)
	} else if f.Raw.ABR != nil && *f.Raw.ABR > 0 {
		bitrateStr = fmt.Sprintf("a%.0fK", *f.Raw.ABR)
	}

	flags := getFlags(f)

	idColored := dkCyan(fmt.Sprintf("%-6s", f.Raw.FormatID))
	resColored := green(fmt.Sprintf("%-8s", f.ResolutionLabel))
	vcodecColored := yellow(fmt.Sprintf("%-6s", vcodec))
	acodecColored := yellow(fmt.Sprintf("%-6s", acodec))
	fpsColored := gray(fmt.Sprintf("%4s", fpsStr))
	sizeColored := cyan(fmt.Sprintf("%10s", sizeStr))
	bitrateColored := colorReset + fmt.Sprintf("%-8s", bitrateStr)
	flagsColored := gray(flags)

	return fmt.Sprintf("%s %s %s %s %s %s %s %s",
		idColored,
		resColored,
		vcodecColored,
		acodecColored,
		fpsColored,
		sizeColored,
		bitrateColored,
		flagsColored,
	)
}

func pickFormats(formats []Format) string {
	if !commandExists("fzf") {
		return pickFormatsFallback(formats)
	}

	header := fmt.Sprintf("%-6s %-8s %-6s %-6s %4s %10s %-8s %s", "ID", "RES", "VCODEC", "ACODEC", "FPS", "SIZE", "BITRATE", "FLAGS")

	var body []string
	formatMap := make(map[string]Format)

	for _, f := range formats {
		body = append(body, formatLine(f))
		formatMap[f.Raw.FormatID] = f
	}

	for {
		fzfArgs := []string{
			"-m",
			"--ansi",
			"--prompt=Select format(s) > ",
			"--header=" + header,
			"--height=50%",
			"--layout=reverse",
			"--border",
			"--bind", "space:toggle+down",
			"--inline-info",
		}

		fzfCmd := exec.Command("fzf", fzfArgs...)
		fzfCmd.Stdin = strings.NewReader(strings.Join(body, "\n"))
		fzfCmd.Stderr = os.Stderr

		selOut, err := fzfCmd.Output()
		if err != nil || strings.TrimSpace(string(selOut)) == "" {
			fmt.Println(red(">> Cancelled."))
			os.Exit(0)
		}

		selectedLines := strings.Split(strings.TrimSpace(string(selOut)), "\n")
		var videoIDs []string
		var audioIDs []string

		for _, line := range selectedLines {
			cleanLine := stripANSI(line)
			parts := strings.Fields(strings.TrimSpace(cleanLine))
			if len(parts) > 0 && parts[0] != "" {
				id := parts[0]
				f := formatMap[id]
				if f.IsVideoOnly {
					videoIDs = append(videoIDs, id)
				} else if f.IsAudioOnly {
					audioIDs = append(audioIDs, id)
				} else {
					videoIDs = append(videoIDs, id) // Muxed treated as video
				}
			}
		}

		if len(videoIDs) > 1 {
			fmt.Println(red(">> Error: Please select only one video track, or a video and an audio track. Try again."))
			continue
		}

		var finalIDs []string
		finalIDs = append(finalIDs, videoIDs...)
		finalIDs = append(finalIDs, audioIDs...)

		formatString := strings.Join(finalIDs, "+")
		fmt.Println(cyan(">> Selected IDs: " + formatString))

		if len(finalIDs) == 1 && len(videoIDs) == 1 && formatMap[videoIDs[0]].IsVideoOnly {
			fmt.Println(yellow(">> Video-only track selected — auto-merging best available audio."))
			formatString = formatString + "+bestaudio/best"
		}

		return formatString
	}
}

func pickFormatsFallback(formats []Format) string {
	fmt.Println(yellow(">> fzf was not found; using the built-in numbered selector."))
	for i, f := range formats {
		fmt.Printf("%3d  %s\n", i+1, stripANSI(formatLine(f)))
	}

	for {
		answer := prompt("Select one video and optionally audio (for example 1 or 1,12; blank cancels): ")
		if answer == "" {
			fmt.Println(yellow(">> Cancelled."))
			os.Exit(0)
		}

		seen := make(map[int]bool)
		var selected []Format
		valid := true
		for _, token := range strings.Split(answer, ",") {
			index, err := strconv.Atoi(strings.TrimSpace(token))
			if err != nil || index < 1 || index > len(formats) || seen[index] {
				valid = false
				break
			}
			seen[index] = true
			selected = append(selected, formats[index-1])
		}
		if !valid || len(selected) == 0 {
			fmt.Println(red(">> Enter valid, comma-separated item numbers."))
			continue
		}

		var videoIDs, audioIDs []string
		for _, f := range selected {
			if f.IsAudioOnly {
				audioIDs = append(audioIDs, f.Raw.FormatID)
			} else {
				videoIDs = append(videoIDs, f.Raw.FormatID)
			}
		}
		if len(videoIDs) > 1 {
			fmt.Println(red(">> Select only one video track, optionally with one or more audio tracks."))
			continue
		}

		ids := append(videoIDs, audioIDs...)
		formatString := strings.Join(ids, "+")
		if len(ids) == 1 && len(videoIDs) == 1 && selected[0].IsVideoOnly {
			formatString += "+bestaudio/best"
		}
		fmt.Println(cyan(">> Selected IDs: " + formatString))
		return formatString
	}
}

// ==============================================================================
// FEATURE 3: MKV MERGE + METADATA EMBEDDING
// ==============================================================================

func runDownload(format string, cookieCmd, trimCmd []string, url string) {
	fmt.Println(cyan(">> Starting Download..."))

	finalArgs := []string{}
	finalArgs = append(finalArgs, clientArgs...)
	finalArgs = append(finalArgs, cookieCmd...)
	finalArgs = append(finalArgs, trimCmd...)
	finalArgs = append(finalArgs, "-P", downloadDir)
	finalArgs = append(finalArgs, "-o", template)

	// Embed options + container options
	finalArgs = append(finalArgs,
		"--merge-output-format", "mkv",
		"--audio-multistreams",
		"--video-multistreams",
		"--embed-chapters",
		"--embed-thumbnail",
		"--convert-thumbnails", "png",
	)

	if strings.TrimSpace(format) != "" {
		finalArgs = append(finalArgs, "-f", format)
	}

	finalArgs = append(finalArgs, url)

	cmd := exec.Command(ytdlp, finalArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		die("Failed to create stdout pipe: " + err.Error())
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		die("yt-dlp failed to start: " + err.Error())
	}

	// Custom stdout filter to clean up noisy yt-dlp output and colorize
	go func() {
		var lineBuf []byte
		buf := make([]byte, 1)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				b := buf[0]
				lineBuf = append(lineBuf, b)
				if b == '\n' || b == '\r' {
					line := string(lineBuf)

					// Filter noisy messages
					if strings.Contains(line, "Downloading webpage") ||
						strings.Contains(line, "Downloading android vr") ||
						strings.Contains(line, "Downloading player") ||
						strings.Contains(line, "Solving JS challenges") ||
						strings.Contains(line, "Downloading m3u8 information") ||
						strings.Contains(line, "Deleting original file") ||
						strings.Contains(line, "There isn't any metadata to add") {
						// Ignore
					} else {
						// Colorize
						if strings.HasPrefix(line, "[download]") {
							line = cyan("[download]") + line[10:]
						} else if strings.HasPrefix(line, "[Merger]") {
							line = yellow("[Merger]") + line[8:]
						} else if strings.HasPrefix(line, "[Metadata]") {
							line = dkCyan("[Metadata]") + line[10:]
						} else if strings.HasPrefix(line, "[ThumbnailsConvertor]") {
							line = gray("[Thumbnails]") + line[21:]
						} else if strings.HasPrefix(line, "[EmbedThumbnail]") {
							line = green("[Thumbnail]") + line[16:]
						} else if strings.HasPrefix(line, "[youtube]") {
							line = gray(line)
						} else if strings.HasPrefix(line, "[info]") {
							line = gray(line)
						}
						os.Stdout.Write([]byte(line))
					}
					lineBuf = lineBuf[:0] // Reset buffer
				}
			}
			if err != nil {
				if len(lineBuf) > 0 {
					os.Stdout.Write(lineBuf)
				}
				break
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		die("yt-dlp exited with error: " + err.Error())
	}
}

// ==============================================================================
// MAIN
// ==============================================================================

func main() {
	refreshPATH()
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
		if home == "" {
			home = "."
		}
	}
	configRoot, configErr := os.UserConfigDir()
	if configErr != nil {
		configRoot = filepath.Join(home, ".config")
	}
	cookiesYT = filepath.Join(configRoot, "vampytd", "cookies-yt.txt")
	cookiesJHS = filepath.Join(configRoot, "vampytd", "cookies-jhs.txt")
	downloadDir = filepath.Join(home, "Downloads", "VampYTD")

	quickMode, quickOptions, trimMode, useCookies, startTime, endTime, url := func() (bool, string, bool, bool, string, string, string) {
		up, qm, qo, tm, uc, st, et, u := parseArgs()
		if up {
			runUpdate()
			os.Exit(0)
		}
		return qm, qo, tm, uc, st, et, u
	}()

	if strings.TrimSpace(url) == "" {
		die("No URL provided.")
	}

	maxRes := 0
	reqCodec := ""
	if quickOptions != "" {
		parts := strings.Split(quickOptions, ",")
		for _, p := range parts {
			p = strings.ToLower(strings.TrimSpace(p))
			if p == "" {
				continue
			}
			resStr := strings.TrimRight(p, "pk")
			if p == "4k" {
				maxRes = 2160
			} else if p == "8k" {
				maxRes = 4320
			} else if p == "2k" {
				maxRes = 1440
			} else if val, err := strconv.Atoi(resStr); err == nil {
				maxRes = val
			} else {
				reqCodec = p
			}
		}
	}

	// Dependencies
	if !commandExists(ytdlp) {
		die("yt-dlp is required and was not found in PATH.")
	}
	if !commandExists(ffmpeg) {
		die("ffmpeg is required for MKV merge and was not found in PATH.")
	}
	if !commandExists("node") {
		die("Node.js is required by the configured yt-dlp JavaScript runtime and was not found in PATH.")
	}

	checkYTDLPAge(configRoot)

	fmt.Println(dkCyan("=== VampYTD ==="))
	fmt.Println(cyan(">> Target: " + url))

	// --- Cookies ---
	var cookieCmd []string
	if useCookies {
		cookieCmd = resolveCookies(url)
	} else {
		fmt.Println(gray(">> [Cookies] DISABLED (Best for 4K/8K detection)"))
	}

	// --- Trim ---
	var trimCmd []string
	if trimMode {
		trimCmd = resolveTrim(&startTime, &endTime)
	}

	// --- Ensure download dir exists ---
	if err := os.MkdirAll(filepath.Clean(downloadDir), 0755); err != nil {
		die("Cannot create download directory: " + err.Error())
	}

	// --- Execute ---
	if quickMode && quickOptions == "" {
		fmt.Println(green(">> [Quick Mode] Auto-selecting best quality..."))
		runDownload("", cookieCmd, trimCmd, url)
	} else {
		meta := fetchMetadata(url, cookieCmd)
		formats := processFormats(meta)
		var formatString string

		if quickMode && quickOptions != "" {
			resStr := "Any"
			if maxRes > 0 {
				resStr = strconv.Itoa(maxRes) + "p"
			}
			codecStr := "Any"
			if reqCodec != "" {
				codecStr = reqCodec
			}
			fmt.Println(cyan(fmt.Sprintf(">> Applying restrictions: Max Res = %s, Codec = %s", resStr, codecStr)))

			var matchingVideo Format
			var matchingAudio Format
			foundVideo := false

			for _, f := range formats {
				if f.IsAudioOnly && matchingAudio.Raw.FormatID == "" {
					matchingAudio = f
				}
			}

			for _, f := range formats {
				if f.IsAudioOnly {
					continue
				}

				if maxRes > 0 {
					h := 0
					if f.Raw.Height != nil {
						h = *f.Raw.Height
					}
					// 20px buffer for weird aspect ratios
					if h > maxRes+20 {
						continue
					}
				}

				if reqCodec != "" {
					if !matchesCodec(f.Raw.VCodec, reqCodec) {
						continue
					}
				}

				// If we haven't found any matching video yet, default to this one
				if !foundVideo {
					matchingVideo = f
					foundVideo = true
					continue
				}

				// Compare resolution and bitrate (effective size) to find the absolute best quality
				currH := 0
				if matchingVideo.Raw.Height != nil {
					currH = *matchingVideo.Raw.Height
				}
				newH := 0
				if f.Raw.Height != nil {
					newH = *f.Raw.Height
				}

				if newH > currH {
					// 1. Prefer higher resolution
					matchingVideo = f
				} else if newH == currH {
					// 2. For same resolution, choose the LARGEST bitrate / filesize
					if f.EffectiveSize > matchingVideo.EffectiveSize {
						matchingVideo = f
					}
				}
			}

			if !foundVideo {
				fmt.Println(red(">> Error: No streams found matching constraints. Opening interactive selector..."))
				formatString = pickFormats(formats)
			} else {
				var finalIDs []string
				finalIDs = append(finalIDs, matchingVideo.Raw.FormatID)
				if matchingVideo.IsVideoOnly && matchingAudio.Raw.FormatID != "" {
					finalIDs = append(finalIDs, matchingAudio.Raw.FormatID)
				}
				formatString = strings.Join(finalIDs, "+")
				fmt.Println(green(">> Found matching format: " + matchingVideo.ResolutionLabel + " " + matchingVideo.Raw.VCodec))
				fmt.Println(cyan(">> Selected IDs: " + formatString))
			}
		} else {
			formatString = pickFormats(formats)
		}

		runDownload(formatString, cookieCmd, trimCmd, url)
	}

	fmt.Println(green(">> Done! Files are in " + downloadDir))
	if isSpawned {
		showToast("VampYTD Download Complete", "Video saved to "+downloadDir)
	}
}
