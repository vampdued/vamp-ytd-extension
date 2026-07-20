package main

import "testing"

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
