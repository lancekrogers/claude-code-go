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
	cc := claude.NewClient("claude")

	// Example 1: Simple text prompt
	fmt.Println("Example 1: Simple text prompt")
	result, err := cc.RunPrompt("Write a function to calculate Fibonacci numbers", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Result:", result.Result)
	fmt.Println()

	// Example 2: JSON output
	fmt.Println("Example 2: JSON output")
	jsonResult, err := cc.RunPrompt("Generate a hello world function", &claude.RunOptions{
		Format: claude.JSONOutput,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cost: $%.6f\n", jsonResult.CostUSD)
	fmt.Printf("Session ID: %s\n", jsonResult.SessionID)
	fmt.Printf("Number of turns: %d\n", jsonResult.NumTurns)
	fmt.Printf("Result: %s\n", jsonResult.Result)
	fmt.Println()

	// Example 3: Custom system prompt
	fmt.Println("Example 3: Custom system prompt")
	customResult, err := cc.RunPrompt("Create a database schema", &claude.RunOptions{
		SystemPrompt: "You are a database architect. Use PostgreSQL best practices and include proper indexing.",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Result:", customResult.Result)
	fmt.Println()

	// Example 4: Reading from a file
	fmt.Println("Example 4: Reading from a file")
	file, err := os.Open("mycode.py")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open file: %v\n", err)
		fmt.Println("Skipping example 4 (no file found)")
	} else {
		defer file.Close()
		fileResult, err := cc.RunFromStdin(file, "Review this code for bugs", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Println("Result:", fileResult.Result)
		}
		fmt.Println()
	}

	// Example 5: Streaming output
	fmt.Println("Example 5: Streaming output")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messageCh, errCh := cc.StreamPrompt(ctx, "Build a React component", &claude.RunOptions{})

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
