// Package dangerous provides access to security-sensitive Claude Code features.
//
// WARNING: This package contains operations that can bypass Claude's safety mechanisms.
// These features should only be used in controlled environments with careful security review.
//
// SECURITY REQUIREMENTS:
//
//   - Must set CLAUDE_ENABLE_DANGEROUS="i-accept-all-risks" environment variable
//   - Cannot be used when NODE_ENV, GO_ENV, or ENVIRONMENT is set to "production"
//   - All operations require explicit confirmation of security risks
//
// DANGEROUS OPERATIONS:
//
//   - BYPASS_ALL_PERMISSIONS: Disables Claude's permission system entirely
//   - SET_ENVIRONMENT_VARIABLES: Injects custom environment variables into Claude process
//   - ENABLE_MCP_DEBUG: Enables detailed MCP debugging that may expose sensitive information
//
// USAGE EXAMPLE:
//
//	// SECURITY REVIEW REQUIRED: Using dangerous Claude client
//	// JUSTIFICATION: Automated testing requires permission bypass
//	// RISK ASSESSMENT: Running in isolated test environment
//	// MITIGATION: Input validated, output logged
//
//	client, err := dangerous.NewDangerousClient("claude")
//	if err != nil {
//	    return err
//	}
//
//	result, err := client.BYPASS_ALL_PERMISSIONS("test prompt", nil)
//
// See SECURITY_SENSITIVE_FEATURES.md for detailed implementation rationale.
package dangerous
