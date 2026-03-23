package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const repo = "AgusRdz/ctx"

type ghRelease struct {
	TagName string `json:"tag_name"`
}

// Run checks for the latest version and updates the binary if needed.
func Run(currentVersion string) {
	fmt.Println("checking for updates...")

	latest, err := latestVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ctx: failed to check for updates: %v\n", err)
		os.Exit(1)
	}

	if latest == currentVersion {
		fmt.Printf("already up to date (%s)\n", currentVersion)
		return
	}

	fmt.Printf("updating %s -> %s\n", currentVersion, latest)

	binaryName := buildBinaryName()

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ctx: failed to find current binary: %v\n", err)
		os.Exit(1)
	}

	// Download and verify to a temp path — never touch the live binary until verified.
	tmpPath := exe + ".new"
	defer os.Remove(tmpPath)
	if err := downloadAndVerify(latest, binaryName, tmpPath); err != nil {
		fmt.Fprintf(os.Stderr, "ctx: update failed: %v\n", err)
		os.Exit(1)
	}

	if err := replaceBinary(exe, tmpPath); err != nil {
		fmt.Fprintf(os.Stderr, "ctx: update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("updated to %s\n", latest)
}

func latestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func buildBinaryName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("ctx-%s-%s%s", goos, goarch, ext)
}

func download(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("ctx: download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("ctx: download returned %d for %s", resp.StatusCode, url)
	}

	// Write to temp file next to the binary, then rename
	tmpPath := destPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("ctx: failed to create temp file: %w", err)
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ctx: failed to write binary: %w", err)
	}

	// On Windows, can't replace a running binary directly.
	// Rename current to .old, rename .tmp to current.
	oldPath := destPath + ".old"
	os.Remove(oldPath)

	if runtime.GOOS == "windows" {
		if err := os.Rename(destPath, oldPath); err != nil && !os.IsNotExist(err) {
			os.Remove(tmpPath)
			return fmt.Errorf("ctx: failed to move old binary: %w", err)
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		if runtime.GOOS == "windows" {
			os.Rename(oldPath, destPath)
		}
		os.Remove(tmpPath)
		return fmt.Errorf("ctx: failed to replace binary: %w", err)
	}

	os.Remove(oldPath) // best-effort on all platforms; ignored if still in use on Windows

	// Verify it's not a 404 HTML page
	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("ctx: failed to verify new binary: %w", err)
	}
	if info.Size() < 1024 {
		return fmt.Errorf("ctx: downloaded file too small (%d bytes), release may not exist", info.Size())
	}

	return nil
}

// maxFetchBytes caps the response size for small text files (checksums, signatures).
const maxFetchBytes = 1 * 1024 * 1024 // 1 MB

// fetchBytes fetches a URL and returns the response body, capped at maxFetchBytes.
func fetchBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ctx: fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ctx: fetch returned %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))
}

// downloadAndVerify downloads a release binary and verifies its integrity:
// 1. Downloads the binary to destPath
// 2. Fetches checksums.txt and checksums.txt.sig for the release
// 3. Verifies the signature on checksums.txt using the embedded public key
// 4. Verifies the binary's SHA256 against checksums.txt
// Removes destPath and returns an error if any step fails.
func downloadAndVerify(version, binaryName, destPath string) error {
	base := fmt.Sprintf("https://github.com/%s/releases/download/%s/", repo, version)

	if err := download(base+binaryName, destPath); err != nil {
		return err
	}

	checksumsTxt, err := fetchBytes(base + "checksums.txt")
	if err != nil {
		os.Remove(destPath)
		return fmt.Errorf("ctx: failed to fetch checksums.txt: %w", err)
	}

	sigBytes, err := fetchBytes(base + "checksums.txt.sig")
	if err != nil {
		os.Remove(destPath)
		return fmt.Errorf("ctx: failed to fetch checksums.txt.sig: %w", err)
	}

	if err := verifySignature(checksumsTxt, strings.TrimSpace(string(sigBytes))); err != nil {
		os.Remove(destPath)
		return err
	}

	if err := verifyBinaryChecksum(destPath, binaryName, checksumsTxt); err != nil {
		os.Remove(destPath)
		return err
	}

	return nil
}

// IsDev returns true if the version looks like a dev build.
func IsDev(version string) bool {
	return version == "dev" || strings.Contains(version, "-dirty")
}
