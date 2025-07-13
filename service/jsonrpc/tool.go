package jsonrpc

// ToolInputSchema is the schema for the tool input.
type ToolInputSchema struct {
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// Tool is a tool that can be called by the server.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema ToolInputSchema `json:"inputSchema"`
}

// ListToolsResult is the result for the list tools method.
type ListToolsResult struct {
	Tools []*Tool `json:"tools"`
}

// CallToolParams is the params for the call tool method.
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Metadata  map[string]interface{} `json:"_meta,omitempty"`
}

type ToolResultContent struct {
	Type     string `json:"type"`               // text, image
	Text     string `json:"text,omitempty"`     // text content
	Data     string `json:"data,omitempty"`     // image content
	MIMEType string `json:"mimeType,omitempty"` // image mime type
}

// CallToolResult is the result for the call tool method.
type CallToolResult struct {
	Content []ToolResultContent `json:"content"`
	IsError bool                `json:"isError,omitempty"`
}
