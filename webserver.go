package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

//go:embed web/*
var webFS embed.FS

type ConvertRequest struct {
	JSONContent string `json:"json_content"`
}

type ConvertResponse struct {
	YAML  string `json:"yaml,omitempty"`
	Error string `json:"error,omitempty"`
}

var (
	activeConnections sync.Map
	shutdownTimer     *time.Timer
	shutdownMutex     sync.Mutex
	lastHeartbeat     int64
)

func startWebServer(port string) error {
	mux := http.NewServeMux()

	// Serve static files
	mux.HandleFunc("/static/", handleStatic)
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/convert", handleConvert)
	mux.HandleFunc("/heartbeat", handleHeartbeat)

	addr := ":" + port
	fmt.Printf("Starting web server on http://localhost%s\n", addr)
	fmt.Printf("Server will automatically shutdown when browser is closed\n")

	// Create HTTP server with connection tracking
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				activeConnections.Store(conn, true)
				cancelShutdownTimer()
			case http.StateClosed, http.StateHijacked:
				activeConnections.Delete(conn)
				scheduleShutdownIfNoConnections()
			}
		},
	}

	// Handle graceful shutdown on signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nReceived shutdown signal...")
		cancel()
		server.Shutdown(context.Background())
	}()

	// Launch browser after a short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost" + addr)
	}()

	// Start shutdown monitoring
	go monitorForAutoShutdown(ctx, server)

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := "web" + r.URL.Path[7:] // Remove "/static" prefix and add "web" prefix

	data, err := webFS.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set appropriate content type
	if len(r.URL.Path) > 4 && r.URL.Path[len(r.URL.Path)-4:] == ".css" {
		w.Header().Set("Content-Type", "text/css")
	} else if len(r.URL.Path) > 3 && r.URL.Path[len(r.URL.Path)-3:] == ".js" {
		w.Header().Set("Content-Type", "application/javascript")
	}

	w.Write(data)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := webFS.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		sendErrorResponse(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	jsonContent := r.FormValue("json_content")

	if jsonContent == "" {
		sendErrorResponse(w, "JSON content is required", http.StatusBadRequest)
		return
	}

	// Convert JSON to YAML using existing function
	yamlResult, err := convertJSONToYAML(jsonContent)
	if err != nil {
		sendErrorResponse(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusBadRequest)
		return
	}

	response := ConvertResponse{
		YAML: yamlResult,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ConvertResponse{
		Error: message,
	}
	json.NewEncoder(w).Encode(response)
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	atomic.StoreInt64(&lastHeartbeat, time.Now().Unix())
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func cancelShutdownTimer() {
	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()

	if shutdownTimer != nil {
		shutdownTimer.Stop()
		shutdownTimer = nil
	}
}

func scheduleShutdownIfNoConnections() {
	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()

	// Count active connections
	count := 0
	activeConnections.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if count == 0 {
		// No active connections, schedule shutdown in 5 seconds
		if shutdownTimer != nil {
			shutdownTimer.Stop()
		}
		shutdownTimer = time.AfterFunc(5*time.Second, func() {
			fmt.Println("No active connections detected. Shutting down server...")
			os.Exit(0)
		})
	}
}

func monitorForAutoShutdown(ctx context.Context, server *http.Server) {
	atomic.StoreInt64(&lastHeartbeat, time.Now().Unix())

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lastBeat := atomic.LoadInt64(&lastHeartbeat)
			if time.Now().Unix()-lastBeat > 5 {
				// No heartbeat for 5 seconds, check if browser is still alive
				fmt.Println("No heartbeat detected for 5 seconds. Browser may have been closed.")

				// Give a short grace period and then shutdown
				time.Sleep(1 * time.Second)

				// Check one more time
				lastBeat = atomic.LoadInt64(&lastHeartbeat)
				if time.Now().Unix()-lastBeat > 6 {
					fmt.Println("Browser appears to be closed. Shutting down server...")
					server.Shutdown(context.Background())
					return
				}
			}
		}
	}
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)

	err := exec.Command(cmd, args...).Start()
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
		fmt.Printf("Please open your browser and navigate to: %s\n", url)
	}
}
