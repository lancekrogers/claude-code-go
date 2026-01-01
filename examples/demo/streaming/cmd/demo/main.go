package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// isExitCommand checks if the input is a command to exit the REPL
func isExitCommand(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	exitCommands := []string{
		"exit", "quit", "bye", "goodbye", "q", ":q", ":quit", ":exit",
		"done", "finish", "end", "stop", "close", "leave", "logout",
		"ctrl+c", "^c", "cancel", "abort", "terminate",
	}

	for _, cmd := range exitCommands {
		if input == cmd {
			return true
		}
	}
	return false
}

// displayStreamingMessage processes and displays streaming messages from Claude
func displayStreamingMessage(msg claude.Message) {
	switch msg.Type {
	case "system":
		if msg.Subtype == "init" {
			fmt.Printf("🔄 Initializing Claude session %s...\n", msg.SessionID[:8])
		}
	case "user":
		// User messages aren't shown as they're the input we provided
	case "assistant":
		// Parse assistant message content
		var content map[string]interface{}
		if err := json.Unmarshal(msg.Message, &content); err == nil {
			if contentArray, ok := content["content"].([]interface{}); ok {
				for _, item := range contentArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if itemMap["type"] == "text" {
							if text, ok := itemMap["text"].(string); ok && strings.TrimSpace(text) != "" {
								fmt.Printf("💬 Claude: %s\n", text)
							}
						} else if itemMap["type"] == "tool_use" {
							if name, ok := itemMap["name"].(string); ok {
								if input, ok := itemMap["input"].(map[string]interface{}); ok {
									displayToolUse(name, input)
								}
							}
						}
					}
				}
			}
		}
	case "tool_result":
		// Tool results are handled by displayToolUse when they appear
	case "result":
		if msg.IsError {
			fmt.Printf("❌ Error: %s\n", msg.Result)
		} else {
			duration := float64(msg.DurationMS) / 1000.0
			fmt.Printf("📊 Response complete - Duration: %.1fs | Turns: %d\n", duration, msg.NumTurns)
		}
	}
}

// displayToolUse shows tool usage in a user-friendly format
func displayToolUse(toolName string, input map[string]interface{}) {
	switch toolName {
	case "Bash":
		if command, ok := input["command"].(string); ok {
			fmt.Printf("🔧 Running: %s\n", command)
		}
	case "Write":
		if path, ok := input["file_path"].(string); ok {
			fmt.Printf("📝 Creating file: %s\n", path)
		}
	case "Edit":
		if path, ok := input["file_path"].(string); ok {
			fmt.Printf("✏️  Editing file: %s\n", path)
		}
	case "Read":
		if path, ok := input["file_path"].(string); ok {
			fmt.Printf("📖 Reading file: %s\n", path)
		}
	case "LS":
		if path, ok := input["path"].(string); ok {
			fmt.Printf("📁 Listing directory: %s\n", path)
		}
	case "Glob":
		if pattern, ok := input["pattern"].(string); ok {
			fmt.Printf("🔍 Searching for: %s\n", pattern)
		}
	default:
		fmt.Printf("🛠️  Using tool: %s\n", toolName)
	}
}

const systemPrompt = `
You are a senior Go engineer with cryptocurrency experience interviewing for a job.

TASK: Create a simple Go program that computes Keccac-256 hashes of files.

EXACT STEPS TO FOLLOW:
1. First run: mkdir it_works
2. Then run: cp examples/demo/streaming/test-file.txt it_works/test-file.txt
3. Create a new file it_works/keccac.go with a Go program
4. Use Go's crypto/sha3 package with sha3.New256() (this IS Keccac-256, not regular SHA-256)
5. Test by running: cd it_works && go run keccac.go test-file.txt
6. Also test: go run keccac.go ../README.md

CRITICAL RESTRICTIONS:
- Work in the current directory, create it_works/ here (NOT in examples/)
- Only create/edit files in it_works/ directory
- Do NOT modify any existing project files
- Do NOT touch go.work, go.mod, or any files outside it_works/

Your program should accept: go run keccac.go <filename>
After completing, briefly explain your approach (≤3 sentences), then ask if you should begin.`

func main() {
	// Create Claude client
	cc := claude.NewClient("claude")

	// First call with system prompt using streaming
	fmt.Println("🚀 Starting streaming demo conversation...")
	fmt.Println("📡 Using real-time tool execution display\n")

	ctx := context.Background()
	messageCh, errCh := cc.StreamPrompt(ctx,
		"In ≤3 sentences, describe your plan and ask if I want you to begin.",
		&claude.RunOptions{
			Format:       claude.StreamJSONOutput,
			SystemPrompt: systemPrompt,
			AllowedTools: []string{
				"Write(it_works/*)",                           // Create files only in it_works/
				"Edit(it_works/*)",                            // Edit files only in it_works/
				"Read(it_works/*)",                            // Read files only in it_works/
				"Read(examples/demo/streaming/test-file.txt)", // Read source test file
				"Read(README.md)",                             // Read README for testing
				"Bash(mkdir it_works*)",                       // Create it_works directory only
				"Bash(chmod it_works/*)",                      // Make files executable in it_works/
				"Bash(cd it_works*)",                          // Change to it_works directory only
				"Bash(cp examples/demo/streaming/test-file.txt it_works/)", // Copy test file to it_works/
				"Bash(go run it_works/*)",                                  // Run Go programs in it_works/
				"Bash(go build it_works/*)",                                // Build Go programs in it_works/
				"Bash(ls it_works*)",                                       // List it_works contents
				"Bash(ls)",                                                 // List current directory
				"Bash(cat it_works/*)",                                     // Display it_works files
				"Bash(echo)",                                               // Create simple test content
				"Bash(pwd)",                                                // Show current directory
			},
		})

	var sessionID string

	// Process initial streaming messages
	for {
		select {
		case msg, ok := <-messageCh:
			if !ok {
				goto repl // Channel closed, start REPL
			}
			displayStreamingMessage(msg)
			if msg.SessionID != "" {
				sessionID = msg.SessionID
			}
			if msg.Type == "result" {
				goto repl // Got final result, start REPL
			}
		case err := <-errCh:
			if err != nil {
				log.Printf("Streaming error: %v", err)
				return
			}
		}
	}

repl:
	// Start REPL loop
	fmt.Println()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			// EOF or error
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			// Blank line, exit
			break
		}

		// Check for exit commands
		if isExitCommand(input) {
			fmt.Println("Goodbye! Thanks for trying the Claude Code Go SDK demo.")
			break
		}

		// Continue conversation with same session and permissions
		// Create a new context with timeout for each request
		requestCtx := context.Background()
		messageCh, errCh := cc.StreamPrompt(requestCtx, input, &claude.RunOptions{
			Format:   claude.StreamJSONOutput,
			ResumeID: sessionID,
			AllowedTools: []string{
				"Write(it_works/*)",                           // Create files only in it_works/
				"Edit(it_works/*)",                            // Edit files only in it_works/
				"Read(it_works/*)",                            // Read files only in it_works/
				"Read(examples/demo/streaming/test-file.txt)", // Read source test file
				"Read(README.md)",                             // Read README for testing
				"Bash(mkdir it_works)",                        // Create it_works directory only
				"Bash(chmod it_works/*)",                      // Make files executable in it_works/
				"Bash(cd it_works/*)",                         // Change to it_works directory only
				"Bash(cp examples/demo/streaming/test-file.txt it_works/)", // Copy test file to it_works/
				"Bash(go run it_works/*)",                                  // Run Go programs in it_works/
				"Bash(go build it_works/*)",                                // Build Go programs in it_works/
				"Bash(ls it_works/*)",                                      // List it_works contents
				"Bash(ls)",                                                 // List current directory
				"Bash(cat it_works/*)",                                     // Display it_works files
				"Bash(echo)",                                               // Create simple test content
				"Bash(pwd)",                                                // Show current directory
			},
		})

		// Process streaming messages for this request
		for {
			select {
			case msg, ok := <-messageCh:
				if !ok {
					goto nextInput // Channel closed, get next input
				}
				displayStreamingMessage(msg)
				if msg.Type == "result" {
					goto nextInput // Got final result, get next input
				}
			case err := <-errCh:
				if err != nil {
					log.Printf("Error: %v", err)
					goto nextInput
				}
			}
		}

	nextInput:
		fmt.Println()
	}

	fmt.Println("Demo completed!")
}

