package claude

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ContentBlockType identifies one block of a structured user message.
type ContentBlockType string

const (
	// ContentBlockText is a plain text block.
	ContentBlockText ContentBlockType = "text"
	// ContentBlockImage is an image block with a base64 source.
	ContentBlockImage ContentBlockType = "image"
)

// ImageSourceBase64 is the only image source type stream-json input accepts.
const ImageSourceBase64 = "base64"

// ContentBlock is one block of a stream-json user message. Build blocks with
// TextBlock and ImageBlock.
type ContentBlock struct {
	Type   ContentBlockType `json:"type"`
	Text   string           `json:"text,omitempty"`
	Source *ImageSource     `json:"source,omitempty"`
}

// ImageSource carries base64-encoded image data and its media type, such as
// image/png.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// TextBlock returns a text content block.
func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: ContentBlockText, Text: text}
}

// ImageBlock returns an image content block that base64-encodes data. It does
// not resize or limit the image; Claude Code and the API apply their own image
// size limits.
func ImageBlock(mediaType string, data []byte) ContentBlock {
	return ContentBlock{
		Type: ContentBlockImage,
		Source: &ImageSource{
			Type:      ImageSourceBase64,
			MediaType: mediaType,
			Data:      base64.StdEncoding.EncodeToString(data),
		},
	}
}

// StreamContent sends one user message made of content blocks, such as text
// and images, and streams the results like StreamPrompt. The message is
// written to stdin as stream-json input, so InputFormat is always
// StreamJSONInput for this call. All other options apply as they do for
// StreamPrompt, including ResumeID. Like StreamPrompt, it uses opts as given
// and does not merge DefaultOptions.Env into opts.Env; RunContentCtx does.
func (c *ClaudeClient) StreamContent(ctx context.Context, blocks []ContentBlock, opts *RunOptions) (<-chan Message, <-chan error) {
	stdin, err := contentStdin(blocks)
	if err != nil {
		return failedStream(err)
	}

	if opts == nil {
		opts = c.DefaultOptions
	}
	streamOpts, err := buildStreamJSONRunOptions(withStreamJSONInput(opts))
	if err != nil {
		return failedStream(err)
	}

	return c.streamMessages(ctx, "", stdin, streamOpts)
}

// RunContent sends one user message made of content blocks and waits for the
// final result.
func (c *ClaudeClient) RunContent(blocks []ContentBlock, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunContentCtx(context.Background(), blocks, opts)
}

// RunContentCtx sends one user message made of content blocks and waits for
// the final result. Like RunPromptCtx, it merges DefaultOptions.Env with
// opts.Env. The run uses stream-json input and output internally.
func (c *ClaudeClient) RunContentCtx(ctx context.Context, blocks []ContentBlock, opts *RunOptions) (*ClaudeResult, error) {
	stdin, err := contentStdin(blocks)
	if err != nil {
		return nil, err
	}

	return c.runPromptWithStructuredHooks(ctx, "", stdin, withStreamJSONInput(c.resolveRunOptions(opts)))
}

func withStreamJSONInput(opts *RunOptions) *RunOptions {
	withInput := cloneRunOptions(opts)
	withInput.InputFormat = StreamJSONInput
	return withInput
}

func contentStdin(blocks []ContentBlock) (io.Reader, error) {
	if err := validateContentBlocks(blocks); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(struct {
		Type    string `json:"type"`
		Message struct {
			Role    string         `json:"role"`
			Content []ContentBlock `json:"content"`
		} `json:"message"`
		ParentToolUseID *string `json:"parent_tool_use_id"`
	}{
		Type: "user",
		Message: struct {
			Role    string         `json:"role"`
			Content []ContentBlock `json:"content"`
		}{Role: "user", Content: blocks},
	})
	if err != nil {
		return nil, newWrappedClaudeError(ErrorValidation, "failed to encode content blocks", err)
	}

	return strings.NewReader(string(payload) + "\n"), nil
}

func validateContentBlocks(blocks []ContentBlock) error {
	if len(blocks) == 0 {
		return NewValidationError("At least one content block is required", "Blocks", 0)
	}

	for i, block := range blocks {
		field := fmt.Sprintf("Blocks[%d]", i)
		switch block.Type {
		case ContentBlockText:
			if strings.TrimSpace(block.Text) == "" {
				return NewValidationError("Text content block cannot be empty", field+".Text", "")
			}
			if block.Source != nil {
				return NewValidationError("Text content block cannot have an image source", field+".Source", block.Source.Type)
			}
		case ContentBlockImage:
			if err := validateImageSource(field, block.Source); err != nil {
				return err
			}
			if block.Text != "" {
				return NewValidationError("Image content block cannot have text", field+".Text", len(block.Text))
			}
		default:
			return NewValidationError("Unsupported content block type", field+".Type", block.Type)
		}
	}

	return nil
}

func validateImageSource(field string, source *ImageSource) error {
	if source == nil {
		return NewValidationError("Image content block requires a source", field+".Source", nil)
	}
	if source.Type != ImageSourceBase64 {
		return NewValidationError("Unsupported image source type", field+".Source.Type", source.Type)
	}
	if strings.TrimSpace(source.MediaType) == "" {
		return NewValidationError("Image content block requires a media type", field+".Source.MediaType", source.MediaType)
	}
	if source.Data == "" {
		return NewValidationError("Image content block requires data", field+".Source.Data", 0)
	}
	if _, err := base64.StdEncoding.DecodeString(source.Data); err != nil {
		return NewValidationError("Image content block data must be standard base64", field+".Source.Data", len(source.Data))
	}
	return nil
}
