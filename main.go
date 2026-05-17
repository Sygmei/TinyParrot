package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultAddr     = ":8080"
	defaultText     = "TinyParrot\n"
	defaultTextFile = "/etc/tinyparrot/text"
)

type config struct {
	addr string
	path string
	text string
}

type response struct {
	status        int
	reason        string
	contentType   string
	allow         string
	contentLength int
	body          string
	writeBody     bool
}

func loadConfig() (config, error) {
	addr := strings.TrimSpace(os.Getenv("TINYPARROT_ADDR"))
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv("PORT"))
		if addr != "" && !strings.Contains(addr, ":") {
			addr = ":" + addr
		}
	}
	if addr == "" {
		addr = defaultAddr
	}

	path := strings.TrimSpace(os.Getenv("TINYPARROT_PATH"))
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	textFile := strings.TrimSpace(os.Getenv("TINYPARROT_TEXT_FILE"))
	if textFile != "" {
		text, err := os.ReadFile(textFile)
		if err != nil {
			return config{}, fmt.Errorf("read TINYPARROT_TEXT_FILE: %w", err)
		}
		return config{addr: addr, path: path, text: string(text)}, nil
	}

	if text, ok := os.LookupEnv("TINYPARROT_TEXT"); ok {
		return config{addr: addr, path: path, text: text}, nil
	}

	if text, err := os.ReadFile(defaultTextFile); err == nil {
		return config{addr: addr, path: path, text: string(text)}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return config{}, fmt.Errorf("read default text file: %w", err)
	}

	return config{addr: addr, path: path, text: defaultText}, nil
}

func parseRequestLine(line string) (method string, target string, ok bool) {
	parts := strings.Fields(line)
	if len(parts) != 3 || !strings.HasPrefix(parts[2], "HTTP/") {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func targetPath(target string) string {
	path, _, _ := strings.Cut(target, "?")
	return path
}

func responseFor(cfg config, method string, target string) response {
	if targetPath(target) != cfg.path {
		return textResponse(404, "Not Found", "not found\n", method != "HEAD")
	}
	if method != "GET" && method != "HEAD" {
		resp := textResponse(405, "Method Not Allowed", "method not allowed\n", method != "HEAD")
		resp.allow = "GET, HEAD"
		return resp
	}

	return response{
		status:        200,
		reason:        "OK",
		contentType:   "text/plain; charset=utf-8",
		contentLength: len(cfg.text),
		body:          cfg.text,
		writeBody:     method == "GET",
	}
}

func textResponse(status int, reason string, body string, writeBody bool) response {
	return response{
		status:        status,
		reason:        reason,
		contentType:   "text/plain; charset=utf-8",
		contentLength: len(body),
		body:          body,
		writeBody:     writeBody,
	}
}

func writeResponse(w io.Writer, resp response) {
	_, _ = io.WriteString(w, "HTTP/1.1 ")
	_, _ = io.WriteString(w, strconv.Itoa(resp.status))
	_, _ = io.WriteString(w, " ")
	_, _ = io.WriteString(w, resp.reason)
	_, _ = io.WriteString(w, "\r\nContent-Type: ")
	_, _ = io.WriteString(w, resp.contentType)
	_, _ = io.WriteString(w, "\r\nCache-Control: no-store\r\nConnection: close\r\nContent-Length: ")
	_, _ = io.WriteString(w, strconv.Itoa(resp.contentLength))
	if resp.allow != "" {
		_, _ = io.WriteString(w, "\r\nAllow: ")
		_, _ = io.WriteString(w, resp.allow)
	}
	_, _ = io.WriteString(w, "\r\n\r\n")
	if resp.writeBody {
		_, _ = io.WriteString(w, resp.body)
	}
}

func handleConn(conn net.Conn, cfg config) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReaderSize(conn, 4096)
	line, err := reader.ReadSlice('\n')
	if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
		return
	}
	if errors.Is(err, bufio.ErrBufferFull) {
		writeResponse(conn, textResponse(400, "Bad Request", "bad request\n", true))
		return
	}

	method, target, ok := parseRequestLine(strings.TrimRight(string(line), "\r\n"))
	if !ok {
		writeResponse(conn, textResponse(400, "Bad Request", "bad request\n", true))
		return
	}
	writeResponse(conn, responseFor(cfg, method, target))
}

func serve(cfg config) error {
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		<-signals
		_ = listener.Close()
	}()

	fmt.Fprintf(os.Stderr, "serving addr=%s path=%s\n", cfg.addr, cfg.path)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			fmt.Fprintf(os.Stderr, "accept: %v\n", err)
			continue
		}
		go handleConn(conn, cfg)
	}
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if err := serve(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}
