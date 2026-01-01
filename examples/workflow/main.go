// Package main demonstrates CI/CD workflow automation with the Claude Code Go SDK.
// This example shows how to:
// 1. Use permission modes for headless automation
// 2. Handle structured output for pipeline integration
// 3. Manage exit codes and errors properly
// 4. Implement batch processing patterns
// 5. Build robust automation workflows
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// =========================================================================
	// Example 1: Headless Automation with Permission Modes
	// =========================================================================
	// In CI/CD, there's no user to approve tool usage.
	// Permission modes allow automated execution.
	fmt.Println("=== Example 1: Permission Modes for CI/CD ===")

	fmt.Println(`Permission modes for automation:

1. PermissionModeDefault (safe for CI)
   - Uses standard permission checks
   - Only approved tools will execute
   - Requires explicit AllowedTools list

2. PermissionModeAcceptEdits (for trusted pipelines)
   - Auto-approves file edit operations
   - Useful for code generation tasks
   - Still respects tool restrictions

3. PermissionModeBypassPermissions (dangerous!)
   - Skips ALL permission checks
   - Should NEVER be used in production CI
   - Reserved for development/testing only`)

	// Example CI/CD configuration
	cc := claude.NewClient("claude")

	ciOpts := &claude.RunOptions{
		Format:         claude.JSONOutput,
		PermissionMode: claude.PermissionModeAcceptEdits,
		AllowedTools: []string{
			"Read",
			"Write",
			"Edit",
			"Grep",
			"Glob",
			"Bash(go build:*)",
			"Bash(go test:*)",
			"Bash(go fmt:*)",
			"Bash(git status:*)",
			"Bash(git diff:*)",
		},
		DisallowedTools: []string{
			"Bash(rm:*)",
			"Bash(sudo:*)",
			"Bash(curl|sh:*)",
		},
		MaxTurns: 10,              // Limit iterations
		Timeout:  5 * time.Minute, // Prevent runaway tasks
	}

	fmt.Println("CI/CD Configuration:")
	fmt.Printf("  Permission Mode: %s\n", ciOpts.PermissionMode)
	fmt.Printf("  Allowed Tools: %v\n", ciOpts.AllowedTools)
	fmt.Printf("  Max Turns: %d\n", ciOpts.MaxTurns)
	fmt.Printf("  Timeout: %v\n", ciOpts.Timeout)
	fmt.Println()

	// =========================================================================
	// Example 2: Structured Output for Pipeline Integration
	// =========================================================================
	// CI/CD pipelines need structured data for decision-making.
	fmt.Println("=== Example 2: Structured Output Handling ===")

	os.Stdout.WriteString(`
Parse JSON output for pipeline integration:

  result, err := client.RunPrompt("Review code for bugs", &claude.RunOptions{
      Format: claude.JSONOutput,
  })
  if err != nil {
      // Handle error (see Example 3)
      os.Exit(1)
  }

  // Access structured data
  fmt.Printf("Session ID: %s\n", result.SessionID)
  fmt.Printf("Cost: $%.6f\n", result.CostUSD)
  fmt.Printf("Duration: %dms\n", result.DurationMS)
  fmt.Printf("Turns: %d\n", result.NumTurns)

  // Check for errors in result
  if result.IsError {
      fmt.Println("Claude reported an error in execution")
      os.Exit(1)
  }

  // Output result for pipeline consumption
  fmt.Println(result.Result)
`)

	// Simulate structured output handling
	simulatedResult := &claude.ClaudeResult{
		Type:          "result",
		Result:        "Code review complete. Found 3 potential issues.",
		CostUSD:       0.00234,
		DurationMS:    4521,
		DurationAPIMS: 4100,
		NumTurns:      3,
		SessionID:     "ci-session-abc123",
		IsError:       false,
	}

	// Output as JSON for pipeline consumption
	jsonOutput, _ := json.MarshalIndent(simulatedResult, "", "  ")
	fmt.Println("Example structured output:")
	fmt.Println(string(jsonOutput))
	fmt.Println()

	// =========================================================================
	// Example 3: Exit Code Management
	// =========================================================================
	// Proper exit codes are critical for CI/CD pipelines.
	fmt.Println("=== Example 3: Exit Code Management ===")

	fmt.Println(`Exit code convention for CI/CD:

  Exit 0: Success
  Exit 1: General error (Claude execution failed)
  Exit 2: Configuration error (invalid options)
  Exit 3: Timeout error
  Exit 4: Permission/auth error
  Exit 5: Rate limit exceeded

Error handling pattern:

  result, err := client.RunPromptCtx(ctx, prompt, opts)
  if err != nil {
      var claudeErr *claude.ClaudeError
      if errors.As(err, &claudeErr) {
          switch claudeErr.Type {
          case claude.ErrorTimeout:
              fmt.Fprintln(os.Stderr, "Timeout:", claudeErr.Message)
              os.Exit(3)
          case claude.ErrorAuthentication:
              fmt.Fprintln(os.Stderr, "Auth error:", claudeErr.Message)
              os.Exit(4)
          case claude.ErrorRateLimit:
              fmt.Fprintln(os.Stderr, "Rate limit:", claudeErr.Message)
              os.Exit(5)
          case claude.ErrorValidation:
              fmt.Fprintln(os.Stderr, "Config error:", claudeErr.Message)
              os.Exit(2)
          default:
              fmt.Fprintln(os.Stderr, "Error:", claudeErr.Message)
              os.Exit(1)
          }
      }
      if errors.Is(err, context.DeadlineExceeded) {
          fmt.Fprintln(os.Stderr, "Context deadline exceeded")
          os.Exit(3)
      }
      fmt.Fprintln(os.Stderr, "Unknown error:", err)
      os.Exit(1)
  }`)
	fmt.Println()

	// =========================================================================
	// Example 4: Batch Processing
	// =========================================================================
	// Process multiple items in a CI pipeline.
	fmt.Println("=== Example 4: Batch Processing ===")

	os.Stdout.WriteString(`
Batch processing pattern for CI:

  type TaskResult struct {
      Name    string
      Success bool
      Result  string
      Cost    float64
      Error   error
  }

  func processBatch(client *claude.ClaudeClient, tasks []string) []TaskResult {
      results := make([]TaskResult, len(tasks))
      ctx := context.Background()

      for i, task := range tasks {
          result, err := client.RunPromptCtx(ctx, task, &claude.RunOptions{
              Format:   claude.JSONOutput,
              MaxTurns: 5,
              Timeout:  2 * time.Minute,
          })

          results[i] = TaskResult{Name: task}
          if err != nil {
              results[i].Error = err
          } else {
              results[i].Success = !result.IsError
              results[i].Result = result.Result
              results[i].Cost = result.CostUSD
          }
      }
      return results
  }

  // Process and report
  tasks := []string{
      "Review main.go for security issues",
      "Check test coverage",
      "Validate API contracts",
  }
  results := processBatch(client, tasks)

  totalCost := 0.0
  failures := 0
  for _, r := range results {
      if r.Error != nil || !r.Success {
          failures++
      }
      totalCost += r.Cost
  }

  fmt.Printf("Processed %d tasks, %d failures, cost: $%.4f\n",
      len(results), failures, totalCost)

  if failures > 0 {
      os.Exit(1)
  }
`)
	fmt.Println()

	// =========================================================================
	// Example 5: GitHub Actions Integration
	// =========================================================================
	// Example of Claude in a GitHub Actions workflow.
	fmt.Println("=== Example 5: GitHub Actions Integration ===")

	fmt.Println(`GitHub Actions workflow example (.github/workflows/claude-review.yml):

  name: Claude Code Review

  on:
    pull_request:
      types: [opened, synchronize]

  jobs:
    review:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4

        - name: Setup Go
          uses: actions/setup-go@v5
          with:
            go-version: '1.21'

        - name: Install Claude CLI
          run: npm install -g @anthropic/claude-cli

        - name: Run Claude Review
          env:
            ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          run: |
            go run ./tools/claude-review/main.go \
              --format json \
              --output review-results.json

        - name: Upload Results
          uses: actions/upload-artifact@v4
          with:
            name: review-results
            path: review-results.json

        - name: Comment on PR
          uses: actions/github-script@v7
          with:
            script: |
              const fs = require('fs');
              const results = JSON.parse(fs.readFileSync('review-results.json'));
              await github.rest.issues.createComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: context.issue.number,
                body: results.summary
              });`)
	fmt.Println()

	// =========================================================================
	// Example 6: Cost Control in CI
	// =========================================================================
	// Managing API costs in automated pipelines.
	fmt.Println("=== Example 6: Cost Control ===")

	os.Stdout.WriteString(`
Budget control for CI pipelines:

  // Create a shared budget tracker for the pipeline
  budgetTracker := claude.NewBudgetTracker()
  budgetTracker.SetMaxBudget(1.00) // $1.00 max for entire pipeline

  // Use the same tracker across all tasks
  opts := &claude.RunOptions{
      Format:        claude.JSONOutput,
      BudgetTracker: budgetTracker,
      MaxBudgetUSD:  0.10, // $0.10 per task limit
  }

  // Run multiple tasks
  for _, task := range tasks {
      if budgetTracker.WouldExceedBudget(0.10) {
          fmt.Println("Budget limit approaching, stopping")
          break
      }

      result, err := client.RunPrompt(task, opts)
      if err != nil {
          // Check if budget exceeded
          if strings.Contains(err.Error(), "budget") {
              fmt.Println("Budget exceeded!")
              break
          }
      }

      fmt.Printf("Task cost: $%.4f, Total: $%.4f\n",
          result.CostUSD, budgetTracker.TotalSpent())
  }

  // Report final cost
  fmt.Printf("Pipeline total cost: $%.4f\n", budgetTracker.TotalSpent())
`)
	fmt.Println()

	// =========================================================================
	// Example 7: Complete CI/CD Tool
	// =========================================================================
	fmt.Println("=== Example 7: Complete CI/CD Tool Pattern ===")

	os.Stdout.WriteString(`Complete CI/CD tool structure:

  package main

  import (
      "context"
      "encoding/json"
      "flag"
      "fmt"
      "os"
      "time"

      "github.com/lancekrogers/claude-code-go/pkg/claude"
  )

  func main() {
      // Parse flags
      prompt := flag.String("prompt", "", "Prompt to execute")
      timeout := flag.Duration("timeout", 5*time.Minute, "Execution timeout")
      maxCost := flag.Float64("max-cost", 0.50, "Maximum cost in USD")
      outputFile := flag.String("output", "", "Output file (JSON)")
      flag.Parse()

      if *prompt == "" {
          fmt.Fprintln(os.Stderr, "Error: -prompt is required")
          os.Exit(2)
      }

      // Setup client
      cc := claude.NewClient("claude")
      ctx, cancel := context.WithTimeout(context.Background(), *timeout)
      defer cancel()

      opts := &claude.RunOptions{
          Format:         claude.JSONOutput,
          PermissionMode: claude.PermissionModeAcceptEdits,
          MaxBudgetUSD:   *maxCost,
          MaxTurns:       10,
          AllowedTools: []string{
              "Read", "Grep", "Glob",
              "Bash(go:*)", "Bash(git:*)",
          },
      }

      // Execute
      result, err := client.RunPromptCtx(ctx, *prompt, opts)
      if err != nil {
          handleError(err)
      }

      // Output
      if *outputFile != "" {
          data, _ := json.MarshalIndent(result, "", "  ")
          os.WriteFile(*outputFile, data, 0644)
      }

      fmt.Println(result.Result)

      if result.IsError {
          os.Exit(1)
      }
  }

  func handleError(err error) {
      var claudeErr *claude.ClaudeError
      if errors.As(err, &claudeErr) {
          fmt.Fprintf(os.Stderr, "Error [%s]: %s\n",
              claudeErr.Type.String(), claudeErr.Message)

          switch claudeErr.Type {
          case claude.ErrorTimeout:
              os.Exit(3)
          case claude.ErrorAuthentication:
              os.Exit(4)
          case claude.ErrorRateLimit:
              os.Exit(5)
          case claude.ErrorValidation:
              os.Exit(2)
          }
      }
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
  }
`)

	// Avoid unused import warnings
	_ = cc
	_ = ciOpts
	_ = os.Stdout
	_ = errors.New
	_ = context.Background
	_ = time.Second
	fmt.Println()

	// =========================================================================
	// Summary
	// =========================================================================
	fmt.Println("=== CI/CD Workflow Summary ===")
	fmt.Println(`Permission Modes:
- PermissionModeDefault       - Safe, explicit tool list required
- PermissionModeAcceptEdits   - Auto-approve edits, good for trusted CI
- PermissionModeBypassPermissions - Dangerous, never in production

Structured Output:
- Use Format: claude.JSONOutput for machine parsing
- Access result.SessionID, CostUSD, DurationMS, NumTurns
- Check result.IsError for execution errors

Exit Codes:
- 0: Success
- 1: General error
- 2: Configuration error
- 3: Timeout
- 4: Authentication error
- 5: Rate limit

Best Practices:
1. Always set MaxTurns to prevent infinite loops
2. Always set Timeout for pipeline predictability
3. Use BudgetTracker to control costs
4. Use AllowedTools to restrict operations
5. Log all costs for billing visibility
6. Handle errors with proper exit codes
7. Output structured data for downstream tools
8. Use ephemeral sessions (NoSessionPersistence) for stateless CI`)
}
