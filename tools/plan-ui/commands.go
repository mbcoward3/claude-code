package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

// runServer runs the HTTP server in the foreground.
func runServer(port int) error {
	srv, ln, err := newServer(port)
	if err != nil {
		return err
	}
	return srv.serve(ln)
}

// runOpen creates/resumes a session and opens the browser.
func runOpen(file string, noOpen bool) error {
	if err := ensureServer(); err != nil {
		return err
	}
	resp, err := apiPost("/api/session", map[string]any{"file": file})
	if err != nil {
		return err
	}
	printJSON(resp)

	if !noOpen {
		if session, ok := resp["session"].(map[string]any); ok {
			if url, ok := session["url"].(string); ok {
				_ = openBrowser(url)
			}
		}
	}
	return nil
}

// runPoll long-polls for human feedback.
func runPoll(file, agentReply string, timeoutMs int) error {
	if err := ensureServer(); err != nil {
		return err
	}
	q := "?file=" + urlQueryEscape(file)
	if agentReply != "" {
		q += "&agent_reply=" + urlQueryEscape(agentReply)
	}
	if timeoutMs > 0 {
		q += "&timeout_ms=" + strconv.Itoa(timeoutMs)
	}
	resp, err := apiGet("/api/poll" + q)
	if err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func runEnd(file string) error {
	if err := ensureServer(); err != nil {
		return err
	}
	resp, err := apiPost("/api/end", map[string]any{"file": file})
	if err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func runStop() error {
	info, err := readServerInfo()
	if err != nil {
		printJSON(map[string]any{"server": map[string]any{"status": "not-running"}})
		return nil
	}
	if _, err := apiPostTo(info.URL, "/api/stop", nil); err != nil {
		printJSON(map[string]any{"server": map[string]any{"status": "not-running"}})
		return nil
	}
	printJSON(map[string]any{"server": map[string]any{"status": "stopped"}})
	return nil
}

func runPlaybook() error {
	fmt.Println(mustAsset("playbook.md"))
	return nil
}

// --- server discovery / spawn ------------------------------------------------

// ensureServer makes sure a healthy background server is running, spawning one
// (detached) if necessary.
func ensureServer() error {
	if info, err := readServerInfo(); err == nil && healthy(info.URL) {
		return nil
	}
	// Spawn `plan-ui server` detached.
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "server")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawning server: %w", err)
	}
	_ = cmd.Process.Release()

	// Wait for the server to come up.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := readServerInfo(); err == nil && healthy(info.URL) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server did not become healthy in time")
}

func healthy(baseURL string) bool {
	client := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(baseURL + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// --- HTTP client helpers -----------------------------------------------------

func baseURL() (string, error) {
	info, err := readServerInfo()
	if err != nil {
		return "", fmt.Errorf("no running server")
	}
	return info.URL, nil
}

func apiPost(path string, body any) (map[string]any, error) {
	base, err := baseURL()
	if err != nil {
		return nil, err
	}
	return apiPostTo(base, path, body)
}

func apiPostTo(base, path string, body any) (map[string]any, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	resp, err := http.Post(base+path, "application/json", &buf)
	if err != nil {
		return nil, err
	}
	return decodeResp(resp)
}

// apiGet uses no client-side timeout so long-polls can wait indefinitely.
func apiGet(path string) (map[string]any, error) {
	base, err := baseURL()
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(base + path)
	if err != nil {
		return nil, err
	}
	return decodeResp(resp)
}

func decodeResp(resp *http.Response) (map[string]any, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("bad server response (%d): %s", resp.StatusCode, string(data))
	}
	if resp.StatusCode >= 400 {
		if msg, ok := out["error"].(string); ok {
			return nil, fmt.Errorf("%s", msg)
		}
	}
	return out, nil
}

// --- misc --------------------------------------------------------------------

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

// portFree reports whether a TCP port is bindable (used by tests/tools).
func portFree(port int) bool {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
