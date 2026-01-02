package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

var exitCommands = []string{
	"exit", "quit", "bye", "goodbye", "q", ":q", ":quit", ":exit",
	"done", "finish", "end", "stop", "close", "leave", "logout",
	"ctrl+c", "^c", "cancel", "abort", "terminate",
}

func isExitCommand(input string) bool {
	return slices.Contains(exitCommands, strings.ToLower(strings.TrimSpace(input)))
}

const systemPrompt = `
ROLE
You are a senior Go engineer with cryptocurrency experience, interviewing for a job.

TASK
1. Create a directory named it_works/ in the current working directory.
2. Copy examples/demo/test-file.txt to it_works/test-file.txt.
3. Inside it_works/, create keccak.go that prints the 256-bit Keccak hash of a file when run:
      go run keccak.go <file>

CONSTRAINTS
• Use ONLY the Go standard library: import "crypto/sha3" and call sha3.SumLegacyKeccak256
  (or sha3.NewLegacyKeccak256). Do NOT use golang.org/x/crypto/sha3 or any other library.
• Work strictly inside it_works/ — do NOT modify, create, or delete files outside that folder.
  Do not touch go.work, go.mod, or any other project files.

AFTER CODING
• cd it_works/ and run:
      go run keccak.go test-file.txt
      go run keccak.go ../README.md
  to show the program works.
• Then output a brief explanation of your approach (no more than three sentences, no bullet lists)
  and ask the interviewer if they would like you to start implementing now.`

func main() {
	// Create Claude client
	cc := claude.NewClient("claude")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// First call with system prompt
	fmt.Println("Starting demo conversation...")
	result, err := cc.RunWithSystemPromptCtx(
		ctx,
		"In <=3 sentences, describe your plan and ask if I want you to begin.",
		systemPrompt,
		&claude.RunOptions{
			Format: claude.JSONOutput,
			AllowedTools: []string{
				"Write(it_works/*)",                                    // Create files only in it_works/
				"Edit(it_works/*)",                                     // Edit files only in it_works/
				"Read(it_works/*)",                                     // Read files only in it_works/
				"Read(examples/demo/basic/test-file.txt)",              // Read source test file
				"Read(README.md)",                                      // Read README for testing
				"Bash(mkdir it_works/*)",                               // Create it_works directory only
				"Bash(chmod it_works/*)",                               // Make files executable in it_works/
				"Bash(cd it_works/*)",                                  // Change to it_works directory only
				"Bash(cp examples/demo/basic/test-file.txt it_works/)", // Copy test file to it_works/
				"Bash(go run it_works/*)",                              // Run Go programs in it_works/
				"Bash(go build it_works/*)",                            // Build Go programs in it_works/
				"Bash(ls it_works/*)",                                  // List it_works contents
				"Bash(ls)",                                             // List current directory
				"Bash(cat it_works/*)",                                 // Display it_works files
				"Bash(echo)",                                           // Create simple test content
				"Bash(pwd)",                                            // Show current directory
			},
		})
	if err != nil {
		log.Fatalf("Error running initial prompt: %v", err)
	}

	// Print Claude's first response
	fmt.Printf("Claude: %s\n\n", result.Result)

	// Capture session ID for conversation continuity
	sessionID := result.SessionID

	// Start REPL loop
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
		requestCtx, requestCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		result, err := cc.RunPromptCtx(requestCtx, input, &claude.RunOptions{
			Format:   claude.JSONOutput,
			ResumeID: sessionID,
			AllowedTools: []string{
				"Write(it_works/*)",                       // Create files only in it_works/
				"Edit(it_works/*)",                        // Edit files only in it_works/
				"Read(it_works/*)",                        // Read files only in it_works/
				"Read(examples/demo/basic/test-file.txt)", // Read source test file
				"Read(README.md)",                         // Read README for testing
				"Bash(mkdir it_works*)",                   // Create it_works directory only
				"Bash(chmod it_works/*)",                  // Make files executable in it_works/
				"Bash(cd it_works*)",                      // Change to it_works directory only
				"Bash(cp examples/demo/basic/test-file.txt it_works/)", // Copy test file to it_works/
				"Bash(go run it_works/*)",                              // Run Go programs in it_works/
				"Bash(go build it_works/*)",                            // Build Go programs in it_works/
				"Bash(ls it_works*)",                                   // List it_works contents
				"Bash(ls)",                                             // List current directory
				"Bash(cat it_works/*)",                                 // Display it_works files
				"Bash(echo)",                                           // Create simple test content
				"Bash(pwd)",                                            // Show current directory
			},
		})
		requestCancel() // Clean up context
		if err != nil {
			log.Printf("Error continuing conversation: %v", err)
			continue
		}

		fmt.Printf("Claude: %s\n\n", result.Result)
	}

	fmt.Println("Demo completed!")
}
