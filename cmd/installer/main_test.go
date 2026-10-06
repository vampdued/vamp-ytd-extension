package main

import (
	"strings"
	"testing"
)

func TestIsUninstallMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		exePath  string
		expected bool
	}{
		{"normal install", []string{}, "C:\\path\\VampYTD-Setup.exe", false},
		{"named uninstall.exe", []string{}, "C:\\path\\uninstall.exe", true},
		{"named Uninstall.EXE uppercase", []string{}, "C:\\path\\Uninstall.EXE", true},
		{"flag --uninstall", []string{"--uninstall"}, "C:\\path\\VampYTD-Setup.exe", true},
		{"flag -uninstall", []string{"-uninstall"}, "C:\\path\\VampYTD-Setup.exe", true},
		{"flag /uninstall", []string{"/uninstall"}, "C:\\path\\VampYTD-Setup.exe", true},
		{"flag -u", []string{"-u"}, "C:\\path\\VampYTD-Setup.exe", true},
		{"other flags", []string{"--quiet", "--force"}, "C:\\path\\VampYTD-Setup.exe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUninstallMode(tt.args, tt.exePath)
			if got != tt.expected {
				t.Errorf("isUninstallMode(%v, %q) = %v; want %v", tt.args, tt.exePath, got, tt.expected)
			}
		})
	}
}

func TestBrowserRegistryKeys(t *testing.T) {
	if len(BrowserRegistryKeys) < 4 {
		t.Fatalf("expected at least 4 browser registry keys, got %d", len(BrowserRegistryKeys))
	}
	for _, key := range BrowserRegistryKeys {
		if !strings.HasPrefix(key, `HKCU\Software\`) {
			t.Errorf("expected HKCU key prefix, got %q", key)
		}
		if !strings.HasSuffix(key, HostName) {
			t.Errorf("expected key to end with HostName %q, got %q", HostName, key)
		}
	}
}

func TestGetInstallDir(t *testing.T) {
	dir, err := getInstallDir()
	if err != nil {
		t.Fatalf("getInstallDir failed: %v", err)
	}
	if !strings.HasSuffix(dir, "VampYTD") {
		t.Errorf("expected install dir to end with VampYTD, got %q", dir)
	}
}
