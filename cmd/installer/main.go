package main

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload/*
var payloadFS embed.FS

const (
	HostName         = "com.vampytd.bridge"
	InstallDirName   = "VampYTD"
	ExtensionDirName = "extension"
)

var BrowserRegistryKeys = []string{
	`HKCU\Software\Google\Chrome\NativeMessagingHosts\` + HostName,
	`HKCU\Software\Microsoft\Edge\NativeMessagingHosts\` + HostName,
	`HKCU\Software\Chromium\NativeMessagingHosts\` + HostName,
	`HKCU\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\` + HostName,
	`HKCU\Software\Vivaldi\NativeMessagingHosts\` + HostName,
}

func enableVTMode() {
	var mode uint32
	h := syscall.Handle(os.Stdout.Fd())
	procGetConsoleMode := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleMode")
	procSetConsoleMode := syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")
	r1, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r1 != 0 {
		_, _, _ = procSetConsoleMode.Call(uintptr(h), uintptr(mode|0x0004))
	}
}

func isUninstallMode(args []string, exePath string) bool {
	base := strings.ToLower(filepath.Base(exePath))
	if strings.Contains(base, "uninstall") {
		return true
	}
	for _, arg := range args {
		low := strings.ToLower(arg)
		if low == "--uninstall" || low == "-uninstall" || low == "/uninstall" || low == "-u" {
			return true
		}
	}
	return false
}

func getInstallDir() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine local app data or home directory: %w", err)
		}
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(localAppData, InstallDirName), nil
}

func printBanner() {
	fmt.Println("\033[1;31m")
	fmt.Println("  ╦  ╦┌─┐┌┬┐┌─┐╦ ╦╔╦╗╔╦╗")
	fmt.Println("  ╚╗╔╝├─┤│││├─┘╚╦╝ ║  ║║")
	fmt.Println("   ╚╝ ┴ ┴┴ ┴┴   ╩  ╩ ═╩╝")
	fmt.Println("  High-Performance YouTube Downloader")
	fmt.Println("\033[0m")
}

func runInstall() error {
	printBanner()
	fmt.Println("\033[1;36mStarting VampYTD Setup for Windows (Chromium)...\033[0m")
	fmt.Println()

	installDir, err := getInstallDir()
	if err != nil {
		return err
	}
	extDir := filepath.Join(installDir, ExtensionDirName)
	bridgePath := filepath.Join(installDir, "bridge.exe")

	// 1. Terminate existing bridge processes
	fmt.Println("\033[1m[1/5] Terminating active bridge processes (if running)...\033[0m")
	killOwnBridge(installDir)
	fmt.Println("  \033[32m✔ Ready for file installation\033[0m")

	// 2. Extract embedded binaries and extension files
	fmt.Printf("\033[1m[2/6] Extracting files to %s...\033[0m\n", installDir)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return fmt.Errorf("failed to create extension directory: %w", err)
	}

	extractedCount := 0
	err = fs.WalkDir(payloadFS, "payload", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "payload" || strings.HasSuffix(path, "placeholder.txt") {
			return nil
		}

		relPath, err := filepath.Rel("payload", path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(installDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := payloadFS.ReadFile(path)
		if err != nil {
			return err
		}

		perm := os.FileMode(0644)
		if strings.HasSuffix(strings.ToLower(destPath), ".exe") {
			perm = 0755
		}
		if err := os.WriteFile(destPath, data, perm); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}
		extractedCount++
		return nil
	})
	if err != nil {
		return fmt.Errorf("payload extraction failed: %w", err)
	}

	// Copy currently running installer as uninstall.exe
	if exePath, err := os.Executable(); err == nil {
		if data, err := os.ReadFile(exePath); err == nil {
			uninstallPath := filepath.Join(installDir, "uninstall.exe")
			_ = os.WriteFile(uninstallPath, data, 0755)
		}
	}
	fmt.Printf("  \033[32m✔ Extracted %d application files successfully\033[0m\n", extractedCount)

	// 3. Install the native messaging host (manifest + browser registrations)
	// via the bridge itself, so registration logic lives in exactly one place.
	fmt.Println("\033[1m[3/5] Installing native messaging host...\033[0m")
	if out, err := exec.Command(bridgePath, "--install-native").CombinedOutput(); err != nil {
		return fmt.Errorf("bridge --install-native failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	fmt.Println("  \033[32m✔ Native messaging host manifest written and browser keys registered\033[0m")

	// 4. Update User PATH
	fmt.Println("\033[1m[4/5] Configuring user environment PATH...\033[0m")
	if err := addDirectoryToUserPath(installDir); err != nil {
		fmt.Printf("  \033[33m⚠ Note: PATH update skipped or manual: %v\033[0m\n", err)
	} else {
		fmt.Printf("  \033[32m✔ Added %s to User PATH\033[0m\n", installDir)
	}

	// 5. Check dependencies
	fmt.Println("\033[1m[5/5] Checking system requirements...\033[0m")
	var missingDeps []string
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		missingDeps = append(missingDeps, "yt-dlp")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		missingDeps = append(missingDeps, "ffmpeg")
	}

	if len(missingDeps) == 0 {
		fmt.Println("  \033[32m✔ All core media tools (yt-dlp, ffmpeg) are installed\033[0m")
	} else {
		fmt.Printf("  \033[33m⚠ Missing optional/required tools: %s\033[0m\n", strings.Join(missingDeps, ", "))
		fmt.Println("    To install with winget, run:")
		fmt.Println("    \033[36mwinget install yt-dlp Gyan.FFmpeg\033[0m")
	}

	// Copy extension folder path to clipboard
	_ = copyPathToClipboard(extDir)
	fmt.Printf("\n  \033[32m✔ Copied extension folder path to clipboard:\033[0m\n    \033[1m%s\033[0m\n", extDir)

	fmt.Println("\n\033[1;32m========================================================\033[0m")
	fmt.Println("\033[1;32m  Installation Complete!\033[0m")
	fmt.Println("\033[1;32m========================================================\033[0m")
	fmt.Println("\nTo activate the extension in Chrome, Edge, Brave, or Vivaldi:")
	fmt.Println("  1. Open \033[36mchrome://extensions\033[0m (or edge://extensions / brave://extensions)")
	fmt.Println("  2. Turn on \033[1mDeveloper mode\033[0m in the top-right corner.")
	fmt.Println("  3. Click \033[1mLoad unpacked\033[0m and select the folder path (already copied to clipboard).")
	fmt.Printf("\nUninstaller is available at: %s\\uninstall.exe\n", installDir)

	// Prompt to open browser & folder
	fmt.Print("\nWould you like to open chrome://extensions and the extension folder now? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	if response == "" || response == "y" || response == "yes" {
		openExtensionsPage()
		openFolder(extDir)
	}

	fmt.Println("\nPress Enter to exit.")
	_, _ = reader.ReadString('\n')
	return nil
}

func runUninstall() error {
	printBanner()
	fmt.Println("\033[1;33mStarting VampYTD Uninstallation...\033[0m")
	fmt.Println()

	installDir, err := getInstallDir()
	if err != nil {
		return err
	}

	// 1. Terminate bridge
	fmt.Println("\033[1m[1/4] Terminating running bridge processes...\033[0m")
	killOwnBridge(installDir)
	fmt.Println("  \033[32m✔ Done\033[0m")

	// 2. Remove registry keys
	fmt.Println("\033[1m[2/4] Removing Chromium browser native messaging registrations...\033[0m")
	for _, key := range BrowserRegistryKeys {
		_ = exec.Command("reg", "delete", key, "/f").Run()
	}
	fmt.Println("  \033[32m✔ Browser registrations removed\033[0m")

	// 3. Remove from PATH
	fmt.Println("\033[1m[3/4] Removing from user environment PATH...\033[0m")
	if err := removeDirectoryFromUserPath(installDir); err != nil {
		fmt.Printf("  \033[33m⚠ Note: PATH cleanup skipped: %v\033[0m\n", err)
	} else {
		fmt.Println("  \033[32m✔ Removed from User PATH\033[0m")
	}

	// 4. Remove install directory
	fmt.Printf("\033[1m[4/4] Removing files from %s...\033[0m\n", installDir)
	// If running inside installDir, schedule directory deletion via background cmd
	exePath, _ := os.Executable()
	if strings.HasPrefix(strings.ToLower(exePath), strings.ToLower(installDir)) {
		// Spawn self-deleting cmd
		delCmd := fmt.Sprintf("timeout /t 1 /nobreak >nul & rmdir /s /q \"%s\"", installDir)
		_ = exec.Command("cmd.exe", "/c", "start", "/b", "cmd.exe", "/c", delCmd).Start()
		fmt.Println("  \033[32m✔ Scheduled directory deletion on exit\033[0m")
	} else {
		_ = os.RemoveAll(installDir)
		fmt.Println("  \033[32m✔ Application directory removed\033[0m")
	}

	fmt.Println("\n\033[1;32mVampYTD has been successfully uninstalled.\033[0m")
	fmt.Println("Press Enter to exit.")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	return nil
}

func addDirectoryToUserPath(dir string) error {
	q := psQuote(dir)
	cmd := fmt.Sprintf(`
		$path = [Environment]::GetEnvironmentVariable('Path', 'User')
		$entries = @($path -split ';' | Where-Object { $_ })
		if ($entries -notcontains '%s') {
			$newPath = ($entries + '%s') -join ';'
			[Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
		}
	`, q, q)
	return exec.Command("powershell", "-NoProfile", "-Command", cmd).Run()
}

func removeDirectoryFromUserPath(dir string) error {
	q := psQuote(dir)
	cmd := fmt.Sprintf(`
		$path = [Environment]::GetEnvironmentVariable('Path', 'User')
		$entries = @($path -split ';' | Where-Object { $_ -and $_ -ne '%s' })
		$newPath = $entries -join ';'
		[Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
	`, q)
	return exec.Command("powershell", "-NoProfile", "-Command", cmd).Run()
}

func copyPathToClipboard(path string) error {
	cmd := fmt.Sprintf("Set-Clipboard -Value '%s'", psQuote(path))
	return exec.Command("powershell", "-NoProfile", "-Command", cmd).Run()
}

// psQuote escapes a string for embedding in a single-quoted PowerShell string.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// killOwnBridge terminates only bridge.exe processes running from our own
// install directory (never an unrelated bridge.exe elsewhere on the system).
func killOwnBridge(installDir string) {
	bridgePath := filepath.Join(installDir, "bridge.exe")
	cmd := fmt.Sprintf(`$procs = Get-CimInstance Win32_Process -Filter "Name = 'bridge.exe'" -ErrorAction SilentlyContinue | Where-Object { $_.ExecutablePath -eq '%s' }; foreach ($p in $procs) { Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue }`, psQuote(bridgePath))
	_ = exec.Command("powershell", "-NoProfile", "-Command", cmd).Run()
}

func openExtensionsPage() {
	browsers := []string{"chrome", "msedge", "brave", "vivaldi"}
	for _, b := range browsers {
		if _, err := exec.LookPath(b); err == nil {
			_ = exec.Command("cmd", "/c", "start", b, "chrome://extensions").Start()
			return
		}
	}
	_ = exec.Command("cmd", "/c", "start", "chrome://extensions").Start()
}

func openFolder(folder string) {
	_ = exec.Command("explorer.exe", folder).Start()
}

func main() {
	enableVTMode()
	exePath, _ := os.Executable()
	if isUninstallMode(os.Args[1:], exePath) {
		if err := runUninstall(); err != nil {
			fmt.Printf("\033[31mUninstallation failed: %v\033[0m\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runInstall(); err != nil {
		fmt.Printf("\033[31mSetup failed: %v\033[0m\n", err)
		os.Exit(1)
	}
}
