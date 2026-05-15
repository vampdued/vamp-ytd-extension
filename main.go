package main

// VampYTD — Go port of ytd.ps1
// Feature Complete implementation according to YTD Core Feature Specification

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ==============================================================================
// CONFIG
// ==============================================================================

const (
	cookiesYT   = `C:\Scripts\cookies-yt.txt`
	cookiesJHS  = `C:\Scripts\cookies-jhs.txt`
	downloadDir = `D:\MEDIA\YT-DL`
	ytdlp       = "yt-dlp"
	ffmpeg      = "ffmpeg"
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

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, red("Error: ")+msg)
	os.Exit(1)
}

func prompt(label string) string {
	fmt.Print(label)
	sc := bufio.NewScanner(os.Stdin)
	sc.Scan()
	return strings.TrimSpace(sc.Text())
}

// ==============================================================================
// ARGUMENT PARSING
// ==============================================================================

func parseArgs() (quickMode bool, quickOptions string, trimMode, useCookies bool, startTime, endTime, url string) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
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
	if strings.HasPrefix(v, "h265") || strings.HasPrefix(v, "hevc") {
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
	quickMode, quickOptions, trimMode, useCookies, startTime, endTime, url := parseArgs()

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
	if !commandExists(ffmpeg) {
		die("ffmpeg is required for MKV merge. Install via Scoop: scoop install ffmpeg")
	}
	if !commandExists("fzf") {
		die("fzf is required. Install via Scoop: scoop install fzf")
	}

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
					if !strings.Contains(strings.ToLower(f.Raw.VCodec), reqCodec) {
						continue
					}
				}

				matchingVideo = f
				foundVideo = true
				break
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
}
