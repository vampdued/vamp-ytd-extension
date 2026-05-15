<#
.SYNOPSIS
    PowerShell port of ytd.sh for Windows 11
.DESCRIPTION
    Wraps yt-dlp with fzf menu, cookie auto-selection, and trimming logic.
#>

# ==============================================================================
# CONFIGURATION
# ==============================================================================

# Cookies (AUTO SELECT BY SITE)
$CookiesYT  = "C:\Scripts\cookies-yt.txt"
$CookiesJHS = "C:\Scripts\cookies-jhs.txt"

# Download Directory
$DownloadDir = "D:\MEDIA\YT-DL"

# Tool Paths
$YTDLP = "yt-dlp"
$FFMPEG = "ffmpeg"

# Player Client Args
$ClientArgs = @("--js-runtime", "node")
# Metadata
$MetaArgs = @("--embed-chapters", "--embed-thumbnail", "--convert-thumbnails", "png")

# Template
$Template = "%(uploader)s - %(title)s [%(id)s] [%(height)sp] [%(vcodec)s] [%(filesize,filesize_approx)S] [%(format_id)s].%(ext)s"

# ==============================================================================
# ARGUMENT PARSING
# ==============================================================================

$QuickMode = $false
$TrimMode = $false
$UseCookies = $false
$StartTime = ""
$EndTime = ""
$URL = ""

for ($i = 0; $i -lt $args.Count; $i++) {
    $arg = $args[$i]
    switch -Regex ($arg) {
        "^(-q|--quick)$" { $QuickMode = $true }
        "^(-c|--cookies)$" { $UseCookies = $true }
        "^(-t|--trim)$" {
            $TrimMode = $true
            if (($i + 1) -lt $args.Count -and $args[$i+1] -match "^\d") {
                $i++
                $StartTime = $args[$i]
                if (($i + 1) -lt $args.Count -and $args[$i+1] -match "^\d") {
                    $i++
                    $EndTime = $args[$i]
                }
            }
        }
        "^(-h|--help)$" {
            Write-Host "Usage: ytd [FLAGS] `"URL`""
            Write-Host "  -q : Quick Mode (Best quality)"
            Write-Host "  -c : Enable Cookies (Auto selects file)"
            Write-Host "  -t : Trim Mode (e.g. -t 8:20 12:20)"
            exit
        }
        default {
            if ($arg -match "^http") { $URL = $arg }
        }
    }
}

if ([string]::IsNullOrWhiteSpace($URL)) {
    Write-Error "Error: No URL provided."
    exit 1
}

if (-not (Get-Command "fzf" -ErrorAction SilentlyContinue)) {
    Write-Error "Error: 'fzf' is missing. Install via Scoop: scoop install fzf"
    exit 1
}

Write-Host ">> Target: $URL" -ForegroundColor Cyan

# ==============================================================================
# LOGIC SETUP
# ==============================================================================

# ------------------------------------------------------------------------------
# 1. Cookies Logic (AUTO SITE DETECTION)
# ------------------------------------------------------------------------------

$CookieCmd = @()

if ($UseCookies) {

    $SelectedCookieFile = $null

    if ($URL -match "hotstar|jiohotstar|hs\.") {
        $SelectedCookieFile = $CookiesJHS
        Write-Host ">> [Cookies] Hotstar Mode (cookies-jhs.txt)" -ForegroundColor Green
    }
    elseif ($URL -match "youtube|youtu\.be") {
        $SelectedCookieFile = $CookiesYT
        Write-Host ">> [Cookies] YouTube Mode (cookies-yt.txt)" -ForegroundColor Green
    }
    else {
        $SelectedCookieFile = $CookiesYT
        Write-Host ">> [Cookies] Default Mode (cookies-yt.txt)" -ForegroundColor DarkCyan
    }

    if ($SelectedCookieFile -and (Test-Path $SelectedCookieFile)) {
        $CookieCmd = @("--cookies", $SelectedCookieFile)
    }
    else {
        Write-Host ">> [Warning] Cookie file missing: $SelectedCookieFile" -ForegroundColor Yellow
    }

} else {
    Write-Host ">> [Cookies] DISABLED (Best for 4K/8K detection)" -ForegroundColor Gray
}

# ------------------------------------------------------------------------------
# 2. Trim Logic
# ------------------------------------------------------------------------------

$TrimCmd = @()
if ($TrimMode) {

    if (-not (Get-Command $FFMPEG -ErrorAction SilentlyContinue)) {
        Write-Error "Error: 'ffmpeg' is missing."
        exit 1
    }

    if ([string]::IsNullOrWhiteSpace($StartTime)) {
        Write-Host "------------------------------------------------"
        Write-Host "TRIM MODE ACTIVE (Enter timestamps)"
        $StartTime = Read-Host "Start Time (e.g. 08:20)"
        $EndTime   = Read-Host "End Time   (e.g. 12:20)"
    }

    if ([string]::IsNullOrWhiteSpace($StartTime) -or [string]::IsNullOrWhiteSpace($EndTime)) {
        Write-Error "Error: Timestamps cannot be empty."
        exit 1
    }

    Write-Host "------------------------------------------------"
    Write-Host ">> [TRIM ENABLED] Cutting: $StartTime to $EndTime"
    Write-Host ">> [MODE] Stream Copy via ffmpeg."
    Write-Host "------------------------------------------------"

    $TrimCmd = @("--download-sections", "*$StartTime-$EndTime", "--downloader", "ffmpeg")
}

# ==============================================================================
# DOWNLOAD EXECUTION
# ==============================================================================

if (-not (Test-Path $DownloadDir)) {
    New-Item -ItemType Directory -Path $DownloadDir | Out-Null
}

function Run-Download ($Format) {

    Write-Host ">> Starting Download..." -ForegroundColor Cyan

    $FinalArgs = @()
    $FinalArgs += $ClientArgs
    $FinalArgs += $MetaArgs
    $FinalArgs += $CookieCmd
    $FinalArgs += $TrimCmd
    $FinalArgs += ("-P", $DownloadDir)
    $FinalArgs += ("-o", $Template)
    $FinalArgs += ("--merge-output-format", "mkv")

    if (-not [string]::IsNullOrWhiteSpace($Format)) {
        $FinalArgs += ("-f", $Format)
    }

    $FinalArgs += $URL

    & $YTDLP $FinalArgs
}

# ------------------------------------------------------------------------------
# MODE EXECUTION
# ------------------------------------------------------------------------------

if ($QuickMode) {

    Write-Host ">> [Quick Mode] Auto-selecting best quality..." -ForegroundColor Green
    Run-Download $null

} else {

    Write-Host ">> Fetching formats..." -ForegroundColor Yellow

    $FetchArgs = $ClientArgs + $CookieCmd + @("-F", $URL)
    $RawList = (& $YTDLP $FetchArgs 2>&1) | ForEach-Object { "$_" }

    $StringList = $RawList | Out-String -Stream
    $Header = $StringList | Select-String -Pattern "ID\s+EXT|format code" | Select-Object -First 1

    $Body = $StringList | Where-Object {
        $_ -notmatch "^\[|ID\s+EXT|format code|WARNING|yt-dlp" -and
        ![string]::IsNullOrWhiteSpace($_)
    }

    if (-not $Body) {
        Write-Error ">> Error: No formats found. Try toggling cookies (-c) or check URL."
        exit 1
    }

    $SelectedLines = $Body | fzf -m `
        --prompt="Select Tracks > " `
        --header="$Header" `
        --height=50% `
        --layout=reverse `
        --border `
        --bind 'space:toggle+down' `
        --inline-info

    if (-not $SelectedLines) {
        Write-Host ">> Cancelled." -ForegroundColor Red
        exit 0
    }

    $SelectedIDs = @()
    foreach ($line in $SelectedLines) {
        $parts = $line.Trim() -split "\s+"
        if ($parts[0]) { $SelectedIDs += $parts[0] }
    }

    $FormatString = $SelectedIDs -join "+"
    Write-Host ">> Selected IDs: $FormatString" -ForegroundColor Cyan

    if ($SelectedIDs.Count -eq 1) {
        if ($SelectedLines -match "video only") {
            Write-Host ">> Video-only track. Auto-merging audio..." -ForegroundColor Yellow
            $FormatString = "${FormatString}+bestaudio/best"
        }
    }

    Run-Download $FormatString
}

Write-Host ">> Done! Files are in $DownloadDir" -ForegroundColor Green