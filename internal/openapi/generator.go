// Package openapi provides OpenAPI tool definition generation.
package openapi

import (
	"log/slog"
)

// Generator generates MCP tool definitions from OpenAPI specs.
type Generator struct {
	logger *slog.Logger
}

// NewGenerator creates a new Generator instance.
func NewGenerator(logger *slog.Logger) *Generator {
	return &Generator{logger: logger}
}

// GenerateTools generates MCP tool definitions from an OpenAPI spec.
func (g *Generator) GenerateTools(spec *Spec) []ToolDefinition {
	g.logger.Debug("Generating tools from spec")

	// TODO: Implement tool generation
	return []ToolDefinition{}
}

// ToolDefinition represents an MCP tool definition.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}
