// Package pathutil centralizes Windows PATH discovery shared by the bridge
// and the ytd downloader: user/machine registry PATH, WinGet links and
// package directories, plus well-known install locations. The filesystem
// scan is expensive, so RefreshPATH re-scans at most once per refreshTTL;
// repeated diagnostics calls within the window are nearly free.
package pathutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procExpandEnvironmentStringsW = modkernel32.NewProc("ExpandEnvironmentStringsW")
)

// refreshTTL bounds how often the expensive PATH re-scan runs.
const refreshTTL = 60 * time.Second

var (
	refreshMu   sync.Mutex
	lastRefresh time.Time
)

// RefreshPATH rebuilds the process PATH from the registry, WinGet
// locations and well-known install directories. On Windows the scan is
// skipped when it already ran within refreshTTL.
func RefreshPATH() {
	if runtime.GOOS != "windows" {
		return
	}
	refreshMu.Lock()
	defer refreshMu.Unlock()
	if time.Since(lastRefresh) < refreshTTL {
		return
	}
	lastRefresh = time.Now()
	refreshPATHNow()
}

func refreshPATHNow() {
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

// CommandAvailable reports whether name resolves in PATH, refreshing the
// PATH first (subject to the refresh TTL).
func CommandAvailable(name string) bool {
	RefreshPATH()
	_, err := exec.LookPath(name)
	return err == nil
}

// DetectJSRuntime probes deno → bun → node in PATH and returns the first
// available runtime's name, or "" when none is found.
func DetectJSRuntime() string {
	for _, candidate := range []string{"deno", "bun", "node"} {
		if CommandAvailable(candidate) {
			return candidate
		}
	}
	return ""
}

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
