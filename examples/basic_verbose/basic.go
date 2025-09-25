package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// Create a new Claude client
	client := claude.NewClient("claude")

	// Example 1: Simple text prompt
	fmt.Println("Example 1: Simple text prompt")
	result, err := client.RunPrompt("Write a function to calculate Fibonacci numbers", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Result:", result.Result)
	fmt.Println()

	// Example 2: JSON output with verbose mode (shows full conversation)
	fmt.Println("Example 2: JSON output with verbose mode")
	jsonResult, err := client.RunPrompt("Generate a hello world function", &claude.RunOptions{
		Format: claude.JSONOutput,
		Verbose: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cost: $%.6f\n", jsonResult.CostUSD)
	fmt.Printf("Session ID: %s\n", jsonResult.SessionID)
	fmt.Printf("Number of turns: %d\n", jsonResult.NumTurns)
	fmt.Printf("Result: %s\n", jsonResult.Result)
	
	// Show all messages from the conversation
	fmt.Printf("Total messages: %d\n", len(jsonResult.Messages))
	for i, msg := range jsonResult.Messages {
		fmt.Printf("Message %d: Type=%s", i+1, msg.Type)
		if msg.Subtype != "" {
			fmt.Printf(", Subtype=%s", msg.Subtype)
		}
		if msg.Result != "" {
			fmt.Printf(", Result=%s", msg.Result)
		}
		fmt.Println()
	}
	fmt.Println()

	// Example 3: Comparing normal vs verbose mode Messages
	fmt.Println("Example 3: Comparing normal vs verbose mode Messages")
	
	// Normal mode - single message
	normalResult, err := client.RunPrompt("Write a simple function", &claude.RunOptions{
		Format: claude.JSONOutput,
		Verbose: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Normal mode - Messages count: %d\n", len(normalResult.Messages))
	fmt.Printf("First message type: %s\n", normalResult.Messages[0].Type)
	
	// Verbose mode - multiple messages  
	verboseResult, err := client.RunPrompt("Write a simple function", &claude.RunOptions{
		Format: claude.JSONOutput,
		Verbose: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Verbose mode - Messages count: %d\n", len(verboseResult.Messages))
	fmt.Println("Message types in verbose mode:")
	for i, msg := range verboseResult.Messages {
		fmt.Printf("  %d. %s", i+1, msg.Type)
		if msg.Subtype != "" {
			fmt.Printf(" (%s)", msg.Subtype)
		}
		fmt.Println()
	}
	fmt.Println()

	// Example 4: Custom system prompt
	fmt.Println("Example 4: Custom system prompt")
	customResult, err := client.RunPrompt("Create a database schema", &claude.RunOptions{
		SystemPrompt: "You are a database architect. Use PostgreSQL best practices and include proper indexing.",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Result:", customResult.Result)
	fmt.Printf("Messages available: %d\n", len(customResult.Messages))
	fmt.Println()

	// Example 5: Reading from a file
	fmt.Println("Example 5: Reading from a file")
	file, err := os.Open("mycode.py")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open file: %v\n", err)
		fmt.Println("Skipping example 5 (no file found)")
	} else {
		defer file.Close()
		fileResult, err := client.RunFromStdin(file, "Review this code for bugs", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Println("Result:", fileResult.Result)
			fmt.Printf("Messages available: %d\n", len(fileResult.Messages))
		}
		fmt.Println()
	}

	// Example 6: Streaming output
	fmt.Println("Example 6: Streaming output")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messageCh, errCh := client.StreamPrompt(ctx, "Build a React component", &claude.RunOptions{})

	go func() {
		for err := range errCh {
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", err)
		}
	}()

	for msg := range messageCh {
		// Convert message to JSON for display
		msgJSON, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Println("Message:", string(msgJSON))

		// If this is a result message, we're done
		if msg.Type == "result" {
			break
		}
	}
}
