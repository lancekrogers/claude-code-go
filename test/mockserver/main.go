package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Mock Claude Code server for testing
func main() {
	port := os.Getenv("MOCK_SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/claude", handleClaude)
	http.HandleFunc("/health", handleHealth)

	fmt.Printf("Mock Claude server starting on port %s\n", port)
	fmt.Println("Endpoints:")
	fmt.Printf("  - POST http://localhost:%s/claude (Mock Claude Code)\n", port)
	fmt.Printf("  - GET  http://localhost:%s/health (Health Check)\n", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"server": "mock-claude",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func handleClaude(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request to understand what kind of response to send
	body := make([]byte, r.ContentLength)
	r.Body.Read(body)

	// Simple request parsing - in real implementation this would be more sophisticated
	requestStr := string(body)
	args := strings.Fields(requestStr)

	// Determine response type based on arguments
	var outputFormat string
	var isStreaming bool

	for i, arg := range args {
		if arg == "--output-format" && i+1 < len(args) {
			outputFormat = args[i+1]
		}
	}

	isStreaming = outputFormat == "stream-json"

	if isStreaming {
		handleStreamingResponse(w, r, args)
	} else {
		handleRegularResponse(w, r, args, outputFormat)
	}
}

func handleRegularResponse(w http.ResponseWriter, r *http.Request, args []string, format string) {
	// Extract prompt from args
	prompt := extractPrompt(args)

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"type":            "result",
			"subtype":         "success",
			"result":          generateMockResponse(prompt),
			"total_cost_usd":  0.001,
			"duration_ms":     1500,
			"duration_api_ms": 1200,
			"is_error":        false,
			"num_turns":       1,
			"session_id":      "mock-session-" + fmt.Sprintf("%d", time.Now().Unix()),
		}
		json.NewEncoder(w).Encode(response)
	} else {
		// Text format
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, generateMockResponse(prompt))
	}
}

func handleStreamingResponse(w http.ResponseWriter, r *http.Request, args []string) {
	prompt := extractPrompt(args)
	sessionID := "mock-stream-" + fmt.Sprintf("%d", time.Now().Unix())

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Create a flusher to send data immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send system init message
	initMsg := map[string]interface{}{
		"type":        "system",
		"subtype":     "init",
		"session_id":  sessionID,
		"tools":       []string{"Bash", "Edit", "Read"},
		"mcp_servers": []interface{}{},
	}
	json.NewEncoder(w).Encode(initMsg)
	flusher.Flush()

	time.Sleep(100 * time.Millisecond)

	// Send assistant message
	response := generateMockResponse(prompt)
	assistantMsg := map[string]interface{}{
		"type":       "assistant",
		"message":    map[string]interface{}{},
		"session_id": sessionID,
		"result":     response,
	}
	json.NewEncoder(w).Encode(assistantMsg)
	flusher.Flush()

	time.Sleep(100 * time.Millisecond)

	// Send final result
	resultMsg := map[string]interface{}{
		"type":            "result",
		"subtype":         "success",
		"total_cost_usd":  0.002,
		"duration_ms":     2000,
		"duration_api_ms": 1800,
		"is_error":        false,
		"num_turns":       1,
		"result":          response,
		"session_id":      sessionID,
	}
	json.NewEncoder(w).Encode(resultMsg)
	flusher.Flush()
}

func extractPrompt(args []string) string {
	// Look for -p flag and extract the prompt
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) {
			// Find the prompt and capture everything until the next flag
			prompt := ""
			for j := i + 1; j < len(args); j++ {
				if strings.HasPrefix(args[j], "-") {
					break
				}
				if prompt != "" {
					prompt += " "
				}
				prompt += args[j]
			}
			return prompt
		}
	}
	return "default prompt"
}

func generateMockResponse(prompt string) string {
	prompt = strings.ToLower(prompt)

	// Generate contextual responses based on prompt content
	if strings.Contains(prompt, "hello") || strings.Contains(prompt, "say") {
		return "Hello! I'm a mock Claude assistant. How can I help you today?"
	}

	if strings.Contains(prompt, "2+2") || strings.Contains(prompt, "math") {
		return "The answer is 4. This is a mock response from the test server."
	}

	if strings.Contains(prompt, "count") {
		return "1, 2, 3, 4, 5. This is a mock counting response."
	}

	if strings.Contains(prompt, "code") || strings.Contains(prompt, "function") {
		return "```go\nfunc mockFunction() {\n    // This is a mock code response\n    fmt.Println(\"Mock function\")\n}\n```"
	}

	if strings.Contains(prompt, "story") {
		return "Once upon a time, in a mock testing environment, there was a simulated Claude assistant that helped developers test their applications..."
	}

	// Default response
	return fmt.Sprintf("This is a mock response to your prompt: '%s'. The mock server is working correctly!", prompt)
}

// MockBinary creates a mock Claude binary that forwards requests to the mock server
func createMockBinary() {
	// This would create a shell script that forwards to our mock server
	// For now, we'll just document the approach
	fmt.Println("To create a mock binary, create a shell script that forwards requests to this server")
}

// Usage instructions
func init() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Mock Claude Server")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  go run test/mockserver/main.go")
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  MOCK_SERVER_PORT - Server port (default: 8080)")
		fmt.Println("")
		fmt.Println("Test with:")
		fmt.Println("  curl -X POST http://localhost:8080/claude -d '-p \"Hello world\"'")
		fmt.Println("  curl -X POST http://localhost:8080/claude -d '-p \"Hello\" --output-format json'")
		fmt.Println("  curl -X POST http://localhost:8080/claude -d '-p \"Count\" --output-format stream-json'")
		os.Exit(0)
	}
}
