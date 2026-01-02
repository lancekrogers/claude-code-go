# Claude Code Go SDK - Streaming Demo Guide

## 🚀 Streaming Demo Overview

This demo showcases the **real-time streaming capabilities** of the Claude Code Go SDK using `StreamJSONOutput` format. You'll see Claude's actions as they happen, providing complete transparency into the AI agent's workflow.

## What You'll Experience

### 🔄 Real-time Tool Execution

```
🔧 Running: mkdir it_works/
📝 Creating file: it_works/keccac.go
🔧 Running: cd it_works/ && go run keccac.go test-file.txt
```

### 💬 Live Assistant Messages

- See Claude's responses stream in real-time
- Watch as each tool is executed step-by-step
- Understand the complete workflow

### 📊 Session Metrics

```
✅ Demo completed successfully!
💰 Cost: $0.0234 | ⏱️ Duration: 12.3s | 🔄 Turns: 4
```

## Demo Workflow

1. **Initialization**: Session setup with streaming enabled
2. **Planning**: Claude explains approach (streamed live)
3. **Execution**: Watch each file operation and command
4. **Testing**: See Keccac hash calculations happen in real-time
5. **Completion**: Final metrics and success confirmation

**Note**: The demo uses Keccac hashing via Go's `crypto/sha3.New256()`, which is the same algorithm used in Ethereum and other cryptocurrencies - not SHA-256.

## Key SDK Features Demonstrated

### StreamJSONOutput Format

```go
messageCh, errCh := client.StreamPrompt(ctx, input, &claude.RunOptions{
    Format: claude.StreamJSONOutput,
    // ... other options
})
```

### Real-time Message Processing

```go
for {
    select {
    case msg, ok := <-messageCh:
        displayStreamingMessage(msg)
    case err := <-errCh:
        // Handle errors
    }
}
```

### Message Type Handling

- **System messages**: Session initialization
- **Assistant messages**: Text responses and tool usage
- **Result messages**: Final completion status and metrics

## Educational Value

This demo teaches you how to:

✅ **Build monitoring dashboards** for AI agent activity  
✅ **Provide real-time feedback** to users  
✅ **Handle streaming JSON responses** properly  
✅ **Parse different message types** from Claude Code  
✅ **Implement professional UIs** with progress tracking  

## Production Use Cases

Perfect for applications that need:

- **Progress bars** showing AI task completion
- **Activity logs** for debugging and monitoring  
- **Real-time dashboards** for team collaboration
- **Transparent AI operations** for user trust

## Interactive Commands

Try these during the demo:

- `"yes, please start coding"` - See the full workflow
- `"can you also test with a larger file?"` - Additional testing
- `"show me the final code"` - Code review
- `"quit"` - Clean exit

## Technical Implementation

The streaming demo demonstrates:

- **Context management** for long-running operations
- **Channel-based communication** for async operations  
- **Error handling** in streaming scenarios
- **Session continuity** across multiple interactions
- **UI formatting** for professional output

---

**🎯 This demo shows the Claude Code Go SDK at its most powerful - perfect for production applications that need real-time AI agent monitoring!**
