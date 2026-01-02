package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lancekrogers/claude-code-go/pkg/claude/dangerous"
)

// SECURITY REVIEW REQUIRED: This example demonstrates dangerous Claude operations
// JUSTIFICATION: Educational example showing proper security practices
// RISK ASSESSMENT: Example code only, not intended for production use
// MITIGATION: Clear warnings, environment checks, controlled input

func main() {
	fmt.Println("🚨 Dangerous Claude Operations Example 🚨")
	fmt.Println("This example demonstrates security-sensitive features.")
	fmt.Println()

	// Check if we're in a safe environment
	if !isDevelopmentEnvironment() {
		log.Fatal("❌ This example can only run in development environment")
	}

	// Example 1: Basic dangerous client creation
	fmt.Println("1. Creating dangerous client...")
	cc, err := createDangerousClient()
	if err != nil {
		log.Fatalf("Failed to create dangerous client: %v", err)
	}
	fmt.Println("✅ Dangerous client created successfully")
	fmt.Println()

	// Example 2: Environment variable injection
	fmt.Println("2. Setting environment variables...")
	if err := demonstrateEnvironmentInjection(cc); err != nil {
		log.Printf("Environment injection failed: %v", err)
	}
	fmt.Println()

	// Example 3: MCP debugging
	fmt.Println("3. Enabling MCP debug mode...")
	if err := cc.ENABLE_MCP_DEBUG(); err != nil {
		log.Printf("MCP debug failed: %v", err)
	} else {
		fmt.Println("✅ MCP debugging enabled")
	}
	fmt.Println()

	// Example 4: Show active security warnings
	fmt.Println("4. Current security warnings:")
	warnings := cc.GetSecurityWarnings()
	for _, warning := range warnings {
		fmt.Printf("⚠️  %s\n", warning)
	}
	if len(warnings) == 0 {
		fmt.Println("No active security bypasses")
	}
	fmt.Println()

	// Example 5: Permission bypass (commented out for safety)
	fmt.Println("5. Permission bypass example (NOT EXECUTED):")
	fmt.Println("   // This would bypass ALL Claude safety controls:")
	fmt.Println("   // result, err := client.BYPASS_ALL_PERMISSIONS(\"dangerous prompt\", nil)")
	fmt.Println("   // WARNING: Only use with trusted, validated input!")
	fmt.Println()

	// Example 6: Clean up
	fmt.Println("6. Resetting dangerous settings...")
	cc.ResetDangerousSettings()
	fmt.Println("✅ All dangerous settings cleared")
	fmt.Println()

	fmt.Println("🎓 Example completed safely!")
	fmt.Println("Remember: These features should only be used when absolutely necessary")
	fmt.Println("and with proper security review and justification.")
}

func isDevelopmentEnvironment() bool {
	// Check if we're in development
	nodeEnv := os.Getenv("NODE_ENV")
	goEnv := os.Getenv("GO_ENV")
	env := os.Getenv("ENVIRONMENT")

	return nodeEnv != "production" && goEnv != "production" && 
		   env != "production" && env != "prod"
}

func createDangerousClient() (*dangerous.DangerousClient, error) {
	// Check if dangerous operations are explicitly enabled
	if os.Getenv("CLAUDE_ENABLE_DANGEROUS") != "i-accept-all-risks" {
		return nil, fmt.Errorf("dangerous operations require CLAUDE_ENABLE_DANGEROUS=i-accept-all-risks")
	}

	return dangerous.NewDangerousClient("claude")
}

func demonstrateEnvironmentInjection(cc *dangerous.DangerousClient) error {
	// Example environment variables (safe for demonstration)
	envVars := map[string]string{
		"DEMO_MODE":        "true",
		"CUSTOM_WORKSPACE": "/tmp/claude-workspace",
		"EXAMPLE_CONFIG":   "development",
	}

	// Also demonstrate warning for sensitive variables
	sensitiveVars := map[string]string{
		"DEMO_SECRET": "not-a-real-secret", // This will trigger a warning
	}

	fmt.Println("Setting safe environment variables...")
	if err := cc.SET_ENVIRONMENT_VARIABLES(envVars); err != nil {
		return err
	}

	fmt.Println("Setting potentially sensitive variables (will show warnings)...")
	if err := cc.SET_ENVIRONMENT_VARIABLES(sensitiveVars); err != nil {
		return err
	}

	fmt.Println("✅ Environment variables configured")
	return nil
}

// Example of how to properly document dangerous operations
func automatedDeploymentExample() error {
	// SECURITY REVIEW REQUIRED: Using dangerous Claude client for deployment
	// JUSTIFICATION: Automated deployment pipeline requires permission bypass
	// RISK ASSESSMENT: 
	//   - Deployment runs in isolated container with limited network access
	//   - Input is validated and comes from trusted CI/CD system
	//   - Output is logged and audited
	// MITIGATION:
	//   - Container has read-only filesystem except for deployment directories
	//   - Network access restricted to deployment targets only
	//   - All operations logged to security audit system
	
	cc, err := dangerous.NewDangerousClient("claude")
	if err != nil {
		return fmt.Errorf("deployment client creation failed: %w", err)
	}

	// Set deployment-specific environment
	deploymentEnv := map[string]string{
		"DEPLOYMENT_MODE": "automated",
		"LOG_LEVEL":       "info",
		"WORKSPACE":       "/deployment/workspace",
	}

	if err := cc.SET_ENVIRONMENT_VARIABLES(deploymentEnv); err != nil {
		return fmt.Errorf("deployment environment setup failed: %w", err)
	}

	// In a real deployment, you would execute the deployment prompt here
	// result, err := client.BYPASS_ALL_PERMISSIONS(deploymentPrompt, &claude.RunOptions{
	//     Format: claude.JSONOutput,
	//     MaxTurns: 10,
	// })

	fmt.Println("Deployment configuration completed (actual deployment not executed in example)")
	return nil
}