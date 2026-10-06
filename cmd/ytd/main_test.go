package main

import (
	"testing"
)

func intPtr(value int) *int             { return &value }
func int64Ptr(value int64) *int64       { return &value }
func float64Ptr(value float64) *float64 { return &value }

func TestMatchesCodecAliases(t *testing.T) {
	tests := []struct {
		codec string
		want  string
	}{
		{codec: "av01.0.08M.08", want: "av1"},
		{codec: "vp09.00.51.08", want: "vp9"},
		{codec: "hev1.1.6.L120", want: "hevc"},
		{codec: "avc1.640028", want: "h264"},
	}
	for _, test := range tests {
		if !matchesCodec(test.codec, test.want) {
			t.Errorf("matchesCodec(%q, %q) = false", test.codec, test.want)
		}
	}
}

func TestProcessFormatsQualityOrder(t *testing.T) {
	metadata := VideoMetadata{Formats: []FormatRaw{
		{FormatID: "audio", VCodec: "none", ACodec: "opus", Filesize: int64Ptr(5)},
		{FormatID: "1080-av1-30", Height: intPtr(1080), FPS: float64Ptr(30), VCodec: "av01.0", ACodec: "none", Filesize: int64Ptr(200)},
		{FormatID: "2160", Height: intPtr(2160), FPS: float64Ptr(30), VCodec: "vp09.0", ACodec: "none", Filesize: int64Ptr(300)},
		{FormatID: "1080-h264-60", Height: intPtr(1080), FPS: float64Ptr(60), VCodec: "avc1.6", ACodec: "none", Filesize: int64Ptr(400)},
	}}

	formats := processFormats(metadata)
	want := []string{"2160", "1080-h264-60", "1080-av1-30", "audio"}
	if len(formats) != len(want) {
		t.Fatalf("got %d formats, want %d", len(formats), len(want))
	}
	for i, id := range want {
		if formats[i].Raw.FormatID != id {
			t.Errorf("position %d = %q, want %q", i, formats[i].Raw.FormatID, id)
		}
	}
}

func TestProcessFormatsFiltersMetadataOnlyEntries(t *testing.T) {
	metadata := VideoMetadata{Formats: []FormatRaw{
		{FormatID: "metadata", VCodec: "none", ACodec: "none"},
		{FormatID: "video", Height: intPtr(720), VCodec: "avc1", ACodec: "none"},
	}}
	formats := processFormats(metadata)
	if len(formats) != 1 || formats[0].Raw.FormatID != "video" {
		t.Fatalf("unexpected filtered formats: %#v", formats)
	}
}

func TestParseArgsFrom(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want parsedArgs
	}{
		{"update short", []string{"-U"}, parsedArgs{updateMode: true}},
		{"update long", []string{"--update"}, parsedArgs{updateMode: true}},
		{"update upgrade", []string{"--upgrade"}, parsedArgs{updateMode: true}},
		{"help short", []string{"-h"}, parsedArgs{showHelp: true}},
		{"help long", []string{"--help"}, parsedArgs{showHelp: true}},
		{"cookies", []string{"-c", "https://youtube.com/watch?v=123"}, parsedArgs{useCookies: true, url: "https://youtube.com/watch?v=123"}},
		{"quick bare", []string{"-q", "https://youtu.be/abc"}, parsedArgs{quickMode: true, url: "https://youtu.be/abc"}},
		{"quick with opts", []string{"-q", "1080p,av1", "https://youtu.be/abc"}, parsedArgs{quickMode: true, quickOptions: "1080p,av1", url: "https://youtu.be/abc"}},
		{"quick equals", []string{"-q=720p", "https://youtu.be/abc"}, parsedArgs{quickMode: true, quickOptions: "720p", url: "https://youtu.be/abc"}},
		{"trim start+end", []string{"-t", "8:20", "12:20", "https://youtu.be/abc"}, parsedArgs{trimMode: true, startTime: "8:20", endTime: "12:20", url: "https://youtu.be/abc"}},
		{"trim start only", []string{"-t", "8:20", "https://youtu.be/abc"}, parsedArgs{trimMode: true, startTime: "8:20", url: "https://youtu.be/abc"}},
		{"dir flag", []string{"--dir", `C:\Vids`, "https://youtu.be/abc"}, parsedArgs{dirFlag: `C:\Vids`, url: "https://youtu.be/abc"}},
		{"dir short", []string{"-d", `/tmp/vids`, "https://youtu.be/abc"}, parsedArgs{dirFlag: `/tmp/vids`, url: "https://youtu.be/abc"}},
		{"dir without value", []string{"--dir"}, parsedArgs{}},
		{"url only", []string{"https://youtube.com/watch?v=123"}, parsedArgs{url: "https://youtube.com/watch?v=123"}},
		{"empty", []string{}, parsedArgs{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseArgsFrom(tt.args); got != tt.want {
				t.Errorf("parseArgsFrom(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseArgsFromSpawned(t *testing.T) {
	isSpawned = false
	pa := parseArgsFrom([]string{"--spawned", "https://youtube.com/watch?v=123"})
	if !isSpawned {
		t.Error("expected isSpawned true for --spawned")
	}
	if pa.url != "https://youtube.com/watch?v=123" {
		t.Errorf("expected url %q, got %q", "https://youtube.com/watch?v=123", pa.url)
	}
	isSpawned = false
}

func TestParseQuickOptions(t *testing.T) {
	tests := []struct {
		input   string
		maxRes  int
		reqCodec string
	}{
		{"1080p", 1080, ""},
		{"720p,av1", 720, "av1"},
		{"1080p,vp9", 1080, "vp9"},
		{"4k", 2160, ""},
		{"8k", 4320, ""},
		{"2k", 1440, ""},
		{"2160", 2160, ""},
		{"av1", 0, "av1"},
		{"hevc", 0, "hevc"},
		{"h264", 0, "h264"},
		{"1080P,AV1", 1080, "av1"},
		{" 720p , hevc ", 720, "hevc"},
		{"", 0, ""},
		{"1080p,,vp9", 1080, "vp9"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			maxRes, reqCodec := parseQuickOptions(tt.input)
			if maxRes != tt.maxRes || reqCodec != tt.reqCodec {
				t.Errorf("parseQuickOptions(%q) = (%d, %q), want (%d, %q)",
					tt.input, maxRes, reqCodec, tt.maxRes, tt.reqCodec)
			}
		})
	}
}

func TestEncodePowerShellCommand(t *testing.T) {
	cmd := "Write-Host 'Hello'"
	encoded := encodePowerShellCommand(cmd)
	if encoded == "" {
		t.Fatal("expected non-empty base64 string")
	}
}

