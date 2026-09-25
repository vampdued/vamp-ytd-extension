package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePayload(t *testing.T) {
	tests := []struct {
		name    string
		payload Payload
		wantErr bool
	}{
		{name: "youtube", payload: Payload{URL: "https://www.youtube.com/watch?v=abc", Mode: "quick-1080p", Codec: "av1"}},
		{name: "short URL", payload: Payload{URL: "https://youtu.be/abc", Mode: "interactive", Codec: "auto"}},
		{name: "hotstar", payload: Payload{URL: "https://www.jiohotstar.com/movies/example", Mode: "quick-max"}},
		{name: "hevc codec", payload: Payload{URL: "https://www.youtube.com/watch?v=abc", Mode: "quick-4k", Codec: "hevc"}},
		{name: "reject HTTP", payload: Payload{URL: "http://youtube.com/watch?v=abc"}, wantErr: true},
		{name: "reject lookalike", payload: Payload{URL: "https://youtube.com.example.test/watch?v=abc"}, wantErr: true},
		{name: "reject mode", payload: Payload{URL: "https://youtube.com/watch?v=abc", Mode: "delete-all"}, wantErr: true},
		{name: "reject codec", payload: Payload{URL: "https://youtube.com/watch?v=abc", Codec: "unknown"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePayload(test.payload)
			if (err != nil) != test.wantErr {
				t.Fatalf("validatePayload() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestExtensionKeyMatchesNativeOrigin(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "VampYTDExtension", "manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(manifest.Key)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(publicKey)
	id := make([]byte, 32)
	for i, value := range hash[:16] {
		id[i*2] = 'a' + value>>4
		id[i*2+1] = 'a' + value&0xf
	}
	wantOrigin := "chrome-extension://" + string(id) + "/"
	if extensionOrigin != wantOrigin {
		t.Fatalf("extension origin = %q, manifest key produces %q", extensionOrigin, wantOrigin)
	}
}

func nativeFrame(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var framed bytes.Buffer
	if err := binary.Write(&framed, binary.LittleEndian, uint32(len(body))); err != nil {
		t.Fatal(err)
	}
	framed.Write(body)
	return framed.Bytes()
}

func readNativeResponse(t *testing.T, reader io.Reader) NativeResponse {
	t.Helper()
	var length uint32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		t.Fatal(err)
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		t.Fatal(err)
	}
	var response NativeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestNativeHostProcessesDownloadRequest(t *testing.T) {
	originalLauncher := launchDownloader
	t.Cleanup(func() { launchDownloader = originalLauncher })
	var gotArgs []string
	launchDownloader = func(_ string, args []string) error {
		gotArgs = append([]string(nil), args...)
		return nil
	}

	input := bytes.NewReader(nativeFrame(t, Payload{
		URL:   "https://www.youtube.com/watch?v=abc",
		Mode:  "quick-1080p",
		Codec: "av1",
	}))
	var output bytes.Buffer
	if err := runNativeHost(input, &output); err != nil {
		t.Fatal(err)
	}
	response := readNativeResponse(t, &output)
	if !response.OK || response.Error != "" {
		t.Fatalf("unexpected native response: %#v", response)
	}
	want := []string{"-q", "1080p,av1", "--spawned", "https://www.youtube.com/watch?v=abc"}
	if len(gotArgs) != len(want) {
		t.Fatalf("launcher args = %#v, want %#v", gotArgs, want)
	}
	for i := range want {
		if gotArgs[i] != want[i] {
			t.Fatalf("launcher args = %#v, want %#v", gotArgs, want)
		}
	}
}

func TestNativeHostPingReturnsDiagnostics(t *testing.T) {
	input := bytes.NewReader(nativeFrame(t, Payload{Action: "ping"}))
	var output bytes.Buffer
	if err := runNativeHost(input, &output); err != nil {
		t.Fatal(err)
	}
	response := readNativeResponse(t, &output)
	if !response.OK || response.Platform == "" || response.DownloadDir == "" {
		t.Fatalf("incomplete diagnostic response: %#v", response)
	}
}

func TestNativeHostRejectsUnsupportedHost(t *testing.T) {
	originalLauncher := launchDownloader
	t.Cleanup(func() { launchDownloader = originalLauncher })
	launchDownloader = func(_ string, _ []string) error {
		t.Fatal("launcher must not be called for an invalid request")
		return nil
	}

	input := bytes.NewReader(nativeFrame(t, Payload{URL: "https://example.com/video"}))
	var output bytes.Buffer
	if err := runNativeHost(input, &output); err != nil {
		t.Fatal(err)
	}
	response := readNativeResponse(t, &output)
	if response.OK || response.Error == "" {
		t.Fatalf("unexpected native response: %#v", response)
	}
}

func TestNativeInvocationDetection(t *testing.T) {
	if !isNativeInvocation([]string{}) ||
		!isNativeInvocation([]string{extensionOrigin}) ||
		!isNativeInvocation([]string{strings.TrimSuffix(extensionOrigin, "/")}) ||
		!isNativeInvocation([]string{firefoxExtensionID}) ||
		!isNativeInvocation([]string{"--native-host"}) {
		t.Fatal("native invocation was not detected")
	}
}

func TestTerminalQuoting(t *testing.T) {
	if got, want := posixShellQuote("a'b c"), `'a'\''b c'`; got != want {
		t.Fatalf("posixShellQuote() = %q, want %q", got, want)
	}
	if got, want := appleScriptEscape(`a\b"c`), `a\\b\"c`; got != want {
		t.Fatalf("appleScriptEscape() = %q, want %q", got, want)
	}
}

func TestBuildWindowsPowerShellScriptPreservesURLArguments(t *testing.T) {
	got := buildWindowsPowerShellScript(
		`C:\Program Files\VampYTD\ytd.exe`,
		[]string{"-q", "1080p,av1", "https://www.youtube.com/watch?v=abc&list=xyz"},
	)
	want := `$arguments = @('-q', '1080p,av1', 'https://www.youtube.com/watch?v=abc&list=xyz'); Start-Process -FilePath 'C:\Program Files\VampYTD\ytd.exe' -ArgumentList $arguments -WorkingDirectory 'C:\Program Files\VampYTD'`
	if got != want {
		t.Fatalf("buildWindowsPowerShellScript() =\n%q\nwant\n%q", got, want)
	}
}

func TestWindowsPathDirIsHostIndependent(t *testing.T) {
	if got, want := windowsPathDir(`C:\VampYTD\ytd.exe`), `C:\VampYTD`; got != want {
		t.Fatalf("windowsPathDir() = %q, want %q", got, want)
	}
}

func TestPowerShellSingleQuoteEscapesApostrophes(t *testing.T) {
	if got, want := powershellSingleQuote("it's"), `'it''s'`; got != want {
		t.Fatalf("powershellSingleQuote() = %q, want %q", got, want)
	}
}

func TestMozillaManifestStructure(t *testing.T) {
	manifest := MozillaNativeHostManifest{
		Name:              nativeHostName,
		Description:       "VampYTD browser integration",
		Path:              `C:\Program Files\VampYTD\bridge.exe`,
		Type:              "stdio",
		AllowedExtensions: []string{firefoxExtensionID},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	exts, ok := decoded["allowed_extensions"].([]any)
	if !ok || len(exts) != 1 || exts[0] != firefoxExtensionID {
		t.Fatalf("allowed_extensions = %#v, want [%s]", decoded["allowed_extensions"], firefoxExtensionID)
	}
	if decoded["name"] != nativeHostName {
		t.Fatalf("name = %v, want %s", decoded["name"], nativeHostName)
	}
}

func TestMozillaManifestPaths(t *testing.T) {
	paths, err := mozillaManifestPaths()
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("expected at least one mozilla manifest path")
	}
	if !strings.HasSuffix(paths[0], ".json") {
		t.Fatalf("expected path to end in .json, got: %s", paths[0])
	}
}
