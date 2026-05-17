package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigUsesTextEnv(t *testing.T) {
	t.Setenv("TINYPARROT_TEXT", "hello")
	t.Setenv("TINYPARROT_TEXT_FILE", "")
	t.Setenv("TINYPARROT_ADDR", ":9090")
	t.Setenv("TINYPARROT_PATH", "message")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.addr != ":9090" {
		t.Fatalf("addr = %q, want :9090", cfg.addr)
	}
	if cfg.path != "/message" {
		t.Fatalf("path = %q, want /message", cfg.path)
	}
	if cfg.text != "hello" {
		t.Fatalf("text = %q, want hello", cfg.text)
	}
}

func TestLoadConfigFileOverridesTextEnv(t *testing.T) {
	textFile := filepath.Join(t.TempDir(), "text")
	if err := os.WriteFile(textFile, []byte("from file\n"), 0o600); err != nil {
		t.Fatalf("write text file: %v", err)
	}
	t.Setenv("TINYPARROT_TEXT", "from env")
	t.Setenv("TINYPARROT_TEXT_FILE", textFile)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.text != "from file\n" {
		t.Fatalf("text = %q, want file content", cfg.text)
	}
}

func TestResponseServesConfiguredText(t *testing.T) {
	resp := responseFor(config{path: "/", text: "squawk"}, "GET", "/")

	if resp.status != 200 {
		t.Fatalf("status = %d, want 200", resp.status)
	}
	if resp.body != "squawk" {
		t.Fatalf("body = %q, want squawk", resp.body)
	}
	if resp.contentType != "text/plain; charset=utf-8" {
		t.Fatalf("contentType = %q, want text/plain; charset=utf-8", resp.contentType)
	}
	if !resp.writeBody {
		t.Fatal("writeBody = false, want true")
	}
}

func TestResponseAllowsQueryString(t *testing.T) {
	resp := responseFor(config{path: "/", text: "squawk"}, "GET", "/?sound=soft")

	if resp.status != 200 {
		t.Fatalf("status = %d, want 200", resp.status)
	}
}

func TestResponseRejectsOtherPaths(t *testing.T) {
	resp := responseFor(config{path: "/message", text: "squawk"}, "GET", "/other")

	if resp.status != 404 {
		t.Fatalf("status = %d, want 404", resp.status)
	}
}

func TestResponseRejectsOtherMethods(t *testing.T) {
	resp := responseFor(config{path: "/", text: "squawk"}, "POST", "/")

	if resp.status != 405 {
		t.Fatalf("status = %d, want 405", resp.status)
	}
	if resp.allow != "GET, HEAD" {
		t.Fatalf("allow = %q, want GET, HEAD", resp.allow)
	}
}

func TestHeadResponseDoesNotWriteBody(t *testing.T) {
	resp := responseFor(config{path: "/", text: "squawk"}, "HEAD", "/")

	if resp.status != 200 {
		t.Fatalf("status = %d, want 200", resp.status)
	}
	if resp.contentLength != len("squawk") {
		t.Fatalf("contentLength = %d, want %d", resp.contentLength, len("squawk"))
	}
	if resp.writeBody {
		t.Fatal("writeBody = true, want false")
	}
}

func TestWriteResponse(t *testing.T) {
	var builder strings.Builder
	writeResponse(&builder, responseFor(config{path: "/", text: "squawk"}, "GET", "/"))

	got := builder.String()
	if !strings.HasPrefix(got, "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response prefix = %q", got)
	}
	if !strings.Contains(got, "\r\nContent-Length: 6\r\n\r\nsquawk") {
		t.Fatalf("response = %q, want content length and body", got)
	}
}
