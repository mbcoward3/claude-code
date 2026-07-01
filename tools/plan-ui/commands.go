package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

// cmdServer runs the HTTP server. With --foreground it blocks; otherwise it is
// intended to be spawned detached by ensureServer.
func cmdServer(args []string) error {
	port := 0
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil {
				port = n
			}
			i++
		}
	}
	srv, ln, err := newServer(port)
	if err != nil {
		return err
	}
	return srv.serve(ln)
}

// cmdOpen creates/resumes a session and opens the browser.
func cmdOpen(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: plan-ui open <file> [--no-open]")
	}
	file := args[0]
	noOpen := hasFlag(args, "--no-open")

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

// cmdPoll long-polls for human feedback.
func cmdPoll(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: plan-ui poll <file> [--agent-reply \"...\"] [--timeout-ms N]")
	}
	file := args[0]
	agentReply := flagValue(args, "--agent-reply")
	timeoutMs := flagValue(args, "--timeout-ms")

	if err := ensureServer(); err != nil {
		return err
	}
	q := "?file=" + urlQueryEscape(file)
	if agentReply != "" {
		q += "&agent_reply=" + urlQueryEscape(agentReply)
	}
	if timeoutMs != "" {
		q += "&timeout_ms=" + urlQueryEscape(timeoutMs)
	}
	resp, err := apiGet("/api/poll" + q)
	if err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func cmdEnd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: plan-ui end <file>")
	}
	if err := ensureServer(); err != nil {
		return err
	}
	resp, err := apiPost("/api/end", map[string]any{"file": args[0]})
	if err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func cmdStop(args []string) error {
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

func cmdPlaybook(args []string) error {
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

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func flagValue(args []string, name string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func urlQueryEscape(s string) string {
	// Minimal escaping sufficient for file paths and short messages.
	var b bytes.Buffer
	for _, r := range s {
		switch {
		case r == ' ':
			b.WriteString("%20")
		case r == '&':
			b.WriteString("%26")
		case r == '?':
			b.WriteString("%3F")
		case r == '#':
			b.WriteString("%23")
		case r == '=':
			b.WriteString("%3D")
		case r == '%':
			b.WriteString("%25")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
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
