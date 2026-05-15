# YTD — Core Feature Specification
## Scope: 4 Features Only

> Language-agnostic. Applies to PowerShell, Python, or Go implementations equally.
> No code. Pure specification, behaviour contracts, edge cases, and decision rules.

---

## Feature 1 — yt-dlp JSON Parser (`-J`)

### Purpose
Extract structured video and format metadata from yt-dlp without parsing human-readable output.

### Command Contract
```
yt-dlp -J --no-playlist <URL>
```

| Flag | Reason |
|------|--------|
| `-J` | Dump full JSON to stdout. Single source of truth. |
| `--no-playlist` | If URL is a playlist link, only fetch the first video. Prevents accidental bulk fetch. |

### What the JSON gives you (fields you must extract)

**Video-level fields:**
| Field | Type | Use |
|-------|------|-----|
| `title` | string | Display, filename template |
| `uploader` | string | Filename template |
| `id` | string | Filename template |
| `duration_string` | string | Display only |
| `thumbnail` | string | Open-in-browser feature |
| `formats` | array | All available format objects |

**Per-format fields (extract all of these):**
| Field | Type | Notes |
|-------|------|-------|
| `format_id` | string | Primary key — pass to `-f` |
| `ext` | string | Container format |
| `height` | int or null | Null = audio-only |
| `width` | int or null | Null = audio-only |
| `fps` | float or null | Null on audio-only or unknown |
| `vcodec` | string | `"none"` = audio-only stream |
| `acodec` | string | `"none"` = video-only stream |
| `filesize` | int or null | Exact bytes — prefer over approx |
| `filesize_approx` | int or null | Fallback when `filesize` is null |
| `dynamic_range` | string or null | `"HDR10"`, `"DV"`, `"SDR"`, or null |
| `format_note` | string or null | Human note (e.g. "Premium", "DASH") |

### Derived fields (compute after parsing, do not trust yt-dlp's labels)

| Derived Field | Rule |
|---------------|------|
| `is_video_only` | `vcodec != "none"` AND `acodec == "none"` |
| `is_audio_only` | `vcodec == "none"` AND `acodec != "none"` |
| `is_muxed` | `vcodec != "none"` AND `acodec != "none"` |
| `effective_size` | `filesize` if not null, else `filesize_approx`, else `0` |
| `resolution_label` | `"${height}p"` if height exists, else `"audio"` |

### Error handling contract

| Condition | Required behaviour |
|-----------|--------------------|
| `yt-dlp` exits non-zero | Abort with message: `"Metadata fetch failed (exit N). Check URL or enable cookies."` |
| stdout is empty | Abort with message: `"yt-dlp returned no data."` |
| JSON parse fails | Abort with message: `"Could not parse yt-dlp output as JSON."` Log raw output to session log. |
| `formats` array is empty or missing | Abort with message: `"No formats found in metadata."` |
| Any per-format field is null | Treat as missing — apply defaults (see Derived fields). Never crash on a null field. |

### What you must NEVER do
- Parse `yt-dlp -F` output (human table)
- Use regex to extract format IDs from any text
- Assume field order or presence — always access by key, always null-check
- Fetch metadata more than once per session for the same URL

---

## Feature 2 — Smart Format Sorting (Resolution → Codec → Size)

### Purpose
Present formats in a consistent, quality-first order that hides yt-dlp's arbitrary internal ordering from the user.

### Sort key — three levels, applied in order

#### Level 1: Resolution (descending)
Sort by `height` descending. Higher resolution first.

| Rule | Detail |
|------|--------|
| Use raw `height` integer | Not the label string — `"1080p"` string sort breaks at `"2160p"` vs `"720p"` |
| Audio-only formats | `height = 0` → always sort to the bottom of the list |
| Unknown height (null) | Treat as `0` — same as audio-only |

Resolution groups (informational — do not hard-code these as categories):
```
2160  → 4K
1440  → 2K / QHD
1080  → Full HD
720   → HD
480   → SD
360   → Low
0     → Audio only
```

#### Level 2: Codec efficiency (ascending rank — lower = better)

Within the same resolution, sort by codec quality. Use this fixed rank table:

| Codec string (vcodec prefix) | Rank | Notes |
|------------------------------|------|-------|
| `av01` | 1 | AV1 — best efficiency |
| `vp9`, `vp09` | 2 | VP9 — good efficiency |
| `h265`, `hevc` | 3 | H.265 — good, widely supported |
| `h264`, `avc1` | 4 | H.264 — universal compatibility |
| anything else | 99 | Unknown codec — sort last within resolution group |

**Matching rule:** Match by prefix, not exact string.
`vcodec = "av01.0.05M.08"` → matches `av01` → rank 1.
`vcodec = "avc1.640028"` → matches `avc1` → rank 4.
`vcodec = "none"` → audio-only → rank 99.

#### Level 3: File size (ascending — smaller first within same res+codec)

Within the same resolution AND codec rank, sort by `effective_size` ascending.
Smaller file = fewer bits for same quality = better encode.

**Why ascending and not descending:**
Two formats at 1080p AV1 — the smaller one is the better encode unless the size difference is extreme. The user can always pick the larger one if they want maximum bitrate.

### Complete sort specification

```
PRIMARY   height DESC          (4K before 1080p before audio)
SECONDARY codec_rank ASC      (AV1 before VP9 before H264)
TERTIARY  effective_size ASC  (smaller before larger, same quality)
```

### Special grouping rules

**Audio-only formats** — always place at the bottom of the list, after all video formats, in their own implicit group. Within audio-only:
- Sort by acodec: `opus` → `aac` → `mp3` → everything else
- Then by effective_size descending (larger audio = higher bitrate = better)

**Muxed formats** (video + audio in one stream) — sort alongside video-only formats using the same three-level key. Do not separate them into their own group.

### Edge cases

| Case | Rule |
|------|------|
| Two formats identical on all three levels | Preserve original yt-dlp order (stable sort) |
| `filesize` and `filesize_approx` both null | `effective_size = 0` — sort to end within group |
| Format has `height` but `vcodec = "none"` | Treat as audio-only regardless of height |
| HDR format vs SDR at same resolution | HDR gets no special rank boost — same sort key. User sees HDR label in display. |

---

## Feature 3 — MKV Merge + Metadata Embedding

### Purpose
Produce a single, self-contained output file with chapters, thumbnail, and no format ambiguity.

### Required yt-dlp flags

| Flag | Value | Reason |
|------|-------|--------|
| `--merge-output-format` | `mkv` | Deterministic container. MKV accepts any codec combination. Avoids `.webm` vs `.mp4` ambiguity. |
| `--embed-chapters` | _(switch)_ | Embeds YouTube chapter markers as MKV chapter atoms. Seekable in any player. |
| `--embed-thumbnail` | _(switch)_ | Embeds video thumbnail as cover art. Visible in file browsers and media players. |
| `--convert-thumbnails` | `png` | Normalises thumbnail format. JPEG thumbnails fail to embed in MKV on some yt-dlp versions. |

### Filename template contract

```
%(uploader)s - %(title)s [%(id)s] [%(height)sp] [%(vcodec)s] [%(format_id)s].%(ext)s
```

| Segment | Source field | Purpose |
|---------|-------------|---------|
| `%(uploader)s` | `uploader` | Channel name |
| `%(title)s` | `title` | Video title |
| `[%(id)s]` | `id` | Unique video ID — prevents collisions |
| `[%(height)sp]` | `height` | Resolution at a glance in filename |
| `[%(vcodec)s]` | `vcodec` | Codec visible without opening file |
| `[%(format_id)s]` | `format_id` | Traceability — which yt-dlp format was selected |
| `.%(ext)s` | `ext` | Always `mkv` after merge |

### Behaviour contracts

| Scenario | Expected behaviour |
|----------|--------------------|
| Single video stream selected (no audio) | yt-dlp auto-merges best audio. Output is still MKV. |
| Audio-only format selected | No merge needed. Output is MKV container with audio stream only. |
| Format already muxed (video+audio) | `--merge-output-format mkv` remuxes to MKV. No re-encode. |
| Thumbnail fetch fails | Download continues. Log warning: `"Thumbnail embed skipped — fetch failed."` Do not abort. |
| Chapter data unavailable | Download continues silently. `--embed-chapters` is a no-op when no chapters exist. |
| Output file already exists | yt-dlp default behaviour applies (appends counter suffix). Do not override this. |

### What must NOT be configurable per-download (config.json only)
- Output container format (`mkv`) — changing this per-download creates inconsistent libraries
- Filename template — must be uniform across all downloads for predictable file management

### What IS configurable per-download
- Whether to embed thumbnail (can be disabled via config flag `embed_thumbnail: false`)
- Whether to embed chapters (can be disabled via config flag `embed_chapters: false`)

### ffmpeg dependency
`--merge-output-format mkv` and `--convert-thumbnails` require ffmpeg.
- Check for ffmpeg at startup using the dependency checker
- If missing: abort with `"ffmpeg is required for MKV merge. Install via Scoop: scoop install ffmpeg"`
- Do not silently fall back to a non-MKV container

---

## Feature 4 — fzf Multi-Select Format Picker

### Purpose
Let the user visually select one or more format tracks using fzf, with keyboard-first interaction and no mouse required.

### Required fzf flags

| Flag | Value | Reason |
|------|-------|--------|
| `-m` | _(switch)_ | Multi-select mode |
| `--prompt` | `"Select format(s) > "` | Clear affordance |
| `--header` | Column header string | Orientation |
| `--height` | `50%` | Leaves terminal context visible above |
| `--layout` | `reverse` | Input at top, list below — more natural |
| `--border` | _(switch)_ | Visual separation from rest of terminal |
| `--bind` | `'space:toggle+down'` | Space selects and advances — crucial for multi-select UX |
| `--inline-info` | _(switch)_ | Shows match count inline, not on a separate line |

### Display line format (fed into fzf)

Each line must be fixed-width columns so fzf's search highlights correctly:

```
{ID,-8} {RES,-8} {VCODEC,-10} {ACODEC,-10} {FPS,4} {SIZE,8}  {FLAGS}
```

| Column | Width | Content | Example |
|--------|-------|---------|---------|
| ID | 8 left | `format_id` | `401     ` |
| RES | 8 left | `resolution_label` | `1080p   ` |
| VCODEC | 10 left | `vcodec` prefix only | `av01      ` |
| ACODEC | 10 left | `acodec` prefix, `"-"` if none | `none      ` |
| FPS | 4 right | `fps` as integer, `"-"` if null | `  60` |
| SIZE | 8 right | human-readable size | `   1.2 GB` |
| FLAGS | variable | space-separated tags | `HDR10 video-only` |

**Header line** (same column widths, passed to `--header`):
```
ID       RES      VCODEC     ACODEC     FPS     SIZE   FLAGS
```

### Size display rules

| Condition | Display |
|-----------|---------|
| `filesize` present | Exact: `1.24 GB` or `823 MB` |
| Only `filesize_approx` present | Approx: `~820 MB` |
| Both null | `?` |

Unit thresholds:
- `≥ 1,000,000,000` bytes → display as GB (1 decimal place)
- `≥ 1,000,000` bytes → display as MB (1 decimal place)
- `< 1,000,000` bytes → display as KB (0 decimal places)

### FLAGS column content rules

| Condition | Tag shown |
|-----------|-----------|
| `is_video_only = true` | `video-only` |
| `is_audio_only = true` | `audio-only` |
| `dynamic_range` is not null and not `"SDR"` | value as-is: `HDR10`, `DV`, `HLG` |
| `format_note` contains `"Premium"` | `premium` |
| No flags apply | _(empty)_ |

### Selection result handling

After fzf exits, process selected lines:

**Step 1 — Extract format IDs**
Split each selected line on whitespace. The first token is always the format ID (column 1, fixed width). Trim whitespace. Collect into a list.

**Step 2 — Build format string**
Join selected IDs with `+`:
- 1 ID selected → `"401"`
- 2 IDs selected → `"401+140"`
- 3+ IDs selected → `"401+140+251"` (yt-dlp accepts this)

**Step 3 — Video-only auto-merge rule**
Condition: exactly 1 ID selected AND that format's `is_video_only = true`

Action: append `+bestaudio/best` to the format string.
Result: `"401+bestaudio/best"`

Log: `"Video-only track selected — auto-merging best available audio."`

**CRITICAL:** Check `is_video_only` on the **format object** (from JSON), NOT by string-matching the fzf output line. The fzf display line is for humans — parse data from the structured source.

**Step 4 — Cancelled selection**
If fzf exits with no selection (user pressed Escape or Ctrl-C):
- Print: `"Cancelled."` in Warning color
- Exit cleanly with code 0 (not an error)

### fzf dependency contract

| Condition | Behaviour |
|-----------|-----------|
| fzf found in PATH | Use fzf as specified |
| fzf not found, fallback enabled in config | Show numbered text list, accept numeric input |
| fzf not found, fallback disabled | Abort: `"fzf is required. Install via Scoop: scoop install fzf"` |

### Numbered fallback display (when fzf unavailable)

```
  1   401      1080p    av01       none        60    1.2 GB   HDR10 video-only
  2   399      1080p    avc1       none        60    980 MB   video-only
  3   140      audio    none       aac          -     12 MB   audio-only

Select format numbers (e.g. 1 3 or 1+3): _
```

Accept: space-separated numbers, or `+`-separated numbers.
Validate: each number must correspond to a displayed index. Re-prompt on invalid input (max 3 attempts, then abort).

---

## Integration Contract — How the 4 Features Connect

```
URL input
    │
    ▼
[Feature 1] yt-dlp -J → parse JSON → extract format objects
    │
    ▼
[Feature 2] Sort formats (res → codec → size)
    │
    ▼
[Feature 4] Build fzf display lines → fzf multi-select → extract IDs from format objects
    │
    ▼
    Resolve format string (+ auto-merge rule)
    │
    ▼
[Feature 3] Start download with MKV flags + metadata embedding
```

### Data flow rules
- Feature 1 output feeds Feature 2 directly — same format object list, sorted in place
- Feature 4 reads display data from Feature 2's sorted list — never re-fetches
- Feature 4 resolves `is_video_only` from Feature 1's parsed objects — never from display strings
- Feature 3 flags are fixed — they do not change based on Feature 4's selection

---

## Open Questions — Confirm Before Coding

1. **fzf fallback:** Should the numbered text fallback be built from day one, or deferred to a later phase? Affects how much abstraction the format-picker layer needs.

2. **Multi video-only selection:** If the user selects two video-only tracks (e.g. one 4K AV1 + one 1080p AVC), should the tool auto-append `+bestaudio/best`, or show an error asking them to pick only one video track?

3. **Audio-only group separator:** Should audio-only formats be visually separated from video formats in the fzf list (e.g. a blank line or a `── audio ──` header row between groups), or just sorted to the bottom with no visual break?

4. **HDR sort preference:** Currently HDR gets no rank boost — it sorts alongside SDR at the same resolution. Should HDR formats be sorted above SDR at the same resolution, making them the default top pick at each resolution level?

5. **Size display in fzf:** When both `filesize` and `filesize_approx` are null, the SIZE column shows `?`. Should those formats be hidden from the fzf list entirely, or always shown?

6. **Thumbnail embed failure:** Spec says continue on thumbnail failure. Should the user see a visible warning in the terminal, or is a log entry sufficient?
