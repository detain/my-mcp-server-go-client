// Package openapi provides OpenAPI tool definition generation.
package openapi

import (
	"log/slog"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/getkin/kin-openapi/openapi3"
)

// Generator generates MCP tool definitions from OpenAPI specs.
type Generator struct {
	logger *slog.Logger
}

// Tool represents an MCP tool definition generated from an OpenAPI operation.
type Tool struct {
	Name        string                 // operationId
	Description string                 // summary + description, truncated to ~900 chars
	HTTPMethod  string                 // GET, POST, PUT, DELETE, PATCH
	Path        string                 // /api/path
	InputSchema map[string]interface{} // JSON schema for tool input
	PathParams  []string               // list of path parameter names
	QueryParams []string               // list of query parameter names
	HasBody     bool                   // whether request has body
	Annotations ToolAnnotations        // MCP 2025-03-26 annotations
}

// ToolAnnotations represents MCP 2025-03-26 tool annotations.
type ToolAnnotations struct {
	Title           string // Human-readable title
	ReadOnlyHint    bool   // True for GET methods (unless destructive GET)
	DestructiveHint bool   // True for DELETE and mutating operations
	IdempotentHint  bool   // True for GET, PUT, DELETE (unless mutating GET)
	OpenWorldHint   bool   // Always true for external APIs
}

// NewGenerator creates a new Generator instance.
func NewGenerator(logger *slog.Logger) *Generator {
	return &Generator{logger: logger}
}

// ExtractTools iterates over all paths in the spec and extracts MCP tool definitions.
func (g *Generator) ExtractTools(spec *openapi3.T) []Tool {
	if spec == nil || spec.Paths == nil || spec.Paths.Len() == 0 {
		g.logger.Warn("No paths found in OpenAPI spec")
		return []Tool{}
	}

	tools := make([]Tool, 0)

	// Iterate over all paths using Map()
	pathMap := spec.Paths.Map()
	for path, pathItem := range pathMap {
		sharedParams := pathItem.Parameters

		// Process each HTTP method using Operations()
		operations := pathItem.Operations()
		for method, operation := range operations {
			if operation == nil {
				continue
			}

			tool := g.buildToolDefinition(path, method, operation, sharedParams, spec)
			if tool != nil {
				tools = append(tools, *tool)
			}
		}
	}

	g.logger.Info("Extracted tools from OpenAPI spec", slog.Int("count", len(tools)))
	return tools
}

// buildToolDefinition converts an OpenAPI operation to a Tool definition.
func (g *Generator) buildToolDefinition(
	path string,
	method string,
	operation *openapi3.Operation,
	sharedParams openapi3.Parameters,
	spec *openapi3.T,
) *Tool {
	if operation == nil {
		return nil
	}

	// Generate or use operationId
	operationId := operation.OperationID
	if operationId == "" {
		operationId = g.generateOperationId(path, method)
	}

	httpMethod := strings.ToUpper(method)

	// Build description from summary and description
	summary := operation.Summary
	description := operation.Description
	toolDescription := summary

	if description != "" && description != summary {
		if toolDescription != "" {
			toolDescription += " — "
		}
		toolDescription += description
	}

	if toolDescription == "" {
		toolDescription = strings.ToUpper(method) + " " + path
	}

	// Inject tag and destructive markers
	tag := ""
	if len(operation.Tags) > 0 {
		tag = operation.Tags[0]
	}

	isDestructive := g.isDestructiveOperation(httpMethod, path, operationId)

	var prefixParts []string
	if tag != "" {
		prefixParts = append(prefixParts, "["+tag+"]")
	}
	if isDestructive {
		prefixParts = append(prefixParts, "[DESTRUCTIVE]")
	}

	if len(prefixParts) > 0 {
		toolDescription = strings.Join(prefixParts, " ") + " " + toolDescription
	}

	// Truncate descriptions to ~900 chars at sentence boundary
	toolDescription = g.truncateDescription(toolDescription, 900)

	// Merge path-level and operation-level parameters
	allParams := g.mergeParameters(sharedParams, operation.Parameters)

	pathParams, queryParams, properties, required := g.extractParameters(allParams, spec)

	// Extract request body schema
	hasBody := false
	bodySchema := g.extractRequestBodySchema(operation, spec)
	if bodySchema != nil {
		hasBody = true
		bodyProps := g.simplifyObjectSchema(bodySchema, spec)
		if props, ok := bodyProps["properties"].(map[string]interface{}); ok {
			for propName, propDef := range props {
				properties[propName] = propDef
			}
		}
		if req, ok := bodyProps["required"].([]interface{}); ok {
			for _, r := range req {
				if rStr, ok := r.(string); ok {
					required = append(required, rStr)
				}
			}
		}
	}

	// Build input schema
	inputSchema := g.buildInputSchema(properties, required)

	// MCP 2025-03-26 tool annotations
	isMutatingGet := httpMethod == "GET" && isDestructive
	annotations := ToolAnnotations{
		Title:           strings.TrimSpace(operation.Summary),
		ReadOnlyHint:    httpMethod == "GET" && !isMutatingGet,
		DestructiveHint: isDestructive,
		IdempotentHint:  g.isIdempotent(httpMethod) && !isMutatingGet,
		OpenWorldHint:   true,
	}

	if annotations.Title == "" {
		annotations.Title = operationId
	}

	return &Tool{
		Name:        operationId,
		Description: toolDescription,
		HTTPMethod:  httpMethod,
		Path:        path,
		InputSchema: inputSchema,
		PathParams:  pathParams,
		QueryParams: queryParams,
		HasBody:     hasBody,
		Annotations: annotations,
	}
}

// mergeParameters combines path-level and operation-level parameters.
func (g *Generator) mergeParameters(shared, operation openapi3.Parameters) openapi3.Parameters {
	if len(shared) == 0 {
		return operation
	}
	if len(operation) == 0 {
		return shared
	}

	// Operation params override shared params with same name/in
	paramMap := make(map[string]*openapi3.Parameter)

	for _, p := range shared {
		if p != nil && p.Value != nil {
			key := p.Value.Name + ":" + p.Value.In
			paramMap[key] = p.Value
		}
	}

	for _, p := range operation {
		if p != nil && p.Value != nil {
			key := p.Value.Name + ":" + p.Value.In
			paramMap[key] = p.Value
		}
	}

	var result openapi3.Parameters
	for _, p := range paramMap {
		result = append(result, &openapi3.ParameterRef{Value: p})
	}

	return result
}

// extractParameters extracts path and query parameters into separate lists and a properties map.
func (g *Generator) extractParameters(params openapi3.Parameters, spec *openapi3.T) ([]string, []string, map[string]interface{}, []string) {
	var pathParams, queryParams []string
	properties := make(map[string]interface{})
	var required []string

	for _, paramRef := range params {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}

		param := paramRef.Value
		paramName := param.Name
		if paramName == "" {
			continue
		}

		// Get parameter schema (SchemaRef -> Schema)
		var paramSchema *openapi3.Schema
		if param.Schema != nil && param.Schema.Value != nil {
			paramSchema = param.Schema.Value
		} else if param.Schema != nil && param.Schema.Ref != "" {
			// Try to resolve ref
			paramSchema = g.resolveSchemaRef(param.Schema.Ref, spec)
		}
		if paramSchema == nil {
			paramSchema = &openapi3.Schema{Type: &openapi3.Types{"string"}}
		}

		propDef := g.simplifySchema(paramSchema)

		// Add parameter description if present
		if param.Description != "" {
			if desc, ok := propDef["description"].(string); ok {
				propDef["description"] = param.Description + ". " + desc
			} else {
				propDef["description"] = param.Description
			}
		}

		paramIn := param.In
		if paramIn == "path" {
			pathParams = append(pathParams, paramName)
			required = append(required, paramName)
		} else if paramIn == "query" {
			queryParams = append(queryParams, paramName)
			if param.Required {
				required = append(required, paramName)
			}
		}
		// Skip header/cookie params

		properties[paramName] = propDef
	}

	return pathParams, queryParams, properties, required
}

// extractRequestBodySchema extracts the schema from requestBody content.
func (g *Generator) extractRequestBodySchema(operation *openapi3.Operation, spec *openapi3.T) *openapi3.Schema {
	if operation.RequestBody == nil {
		return nil
	}

	// Handle RequestBodyRef
	var requestBody *openapi3.RequestBody
	if operation.RequestBody.Ref != "" {
		requestBody = g.resolveRequestBodyRefValue(operation.RequestBody.Ref, spec)
	} else if operation.RequestBody.Value != nil {
		requestBody = operation.RequestBody.Value
	}

	if requestBody == nil {
		return nil
	}

	content := requestBody.Content

	// Try application/json first, then multipart/form-data
	for _, mediaType := range []string{"application/json", "multipart/form-data"} {
		mt, ok := content[mediaType]
		if !ok {
			continue
		}

		// Get schema from MediaType (SchemaRef -> Schema)
		if mt.Schema != nil {
			if mt.Schema.Value != nil {
				return mt.Schema.Value
			}
			if mt.Schema.Ref != "" {
				return g.resolveSchemaRef(mt.Schema.Ref, spec)
			}
		}
	}

	return nil
}

// resolveSchemaRef resolves a $ref string to a Schema.
func (g *Generator) resolveSchemaRef(ref string, spec *openapi3.T) *openapi3.Schema {
	resolved := g.resolveRef(ref, spec)
	if resolvedSchema, ok := resolved.(*openapi3.Schema); ok {
		return resolvedSchema
	}
	return nil
}

// resolveRequestBodyRefValue resolves a $ref string to a RequestBody.
func (g *Generator) resolveRequestBodyRefValue(ref string, spec *openapi3.T) *openapi3.RequestBody {
	resolved := g.resolveRef(ref, spec)
	if resolvedReq, ok := resolved.(*openapi3.RequestBody); ok {
		return resolvedReq
	}
	return nil
}

// resolveRef resolves a #/... reference path.
func (g *Generator) resolveRef(ref string, spec *openapi3.T) interface{} {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}

	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	current := interface{}(spec)

	for _, part := range parts {
		// Unescape ~1 -> / and ~0 -> ~
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")

		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else if s, ok := current.(*openapi3.T); ok {
			switch part {
			case "paths":
				current = s.Paths
			case "components":
				current = s.Components
			case "schemas":
				if s.Components != nil {
					current = s.Components.Schemas
				}
			case "parameters":
				if s.Components != nil {
					current = s.Components.Parameters
				}
			case "requestBodies":
				if s.Components != nil {
					current = s.Components.RequestBodies
				}
			default:
				return nil
			}
		} else if schemas, ok := current.(openapi3.Schemas); ok {
			if schema, exists := schemas[part]; exists {
				if schema.Value != nil {
					return schema.Value
				}
				return nil
			}
			return nil
		} else if params, ok := current.(openapi3.ParametersMap); ok {
			if param, exists := params[part]; exists {
				if param.Value != nil {
					return param.Value
				}
				return nil
			}
			return nil
		} else if reqBodies, ok := current.(openapi3.RequestBodies); ok {
			if reqBody, exists := reqBodies[part]; exists {
				if reqBody.Value != nil {
					return reqBody.Value
				}
				return nil
			}
			return nil
		} else if m, ok := current.(map[string]*openapi3.SchemaRef); ok {
			if schemaRef, exists := m[part]; exists {
				if schemaRef.Value != nil {
					return schemaRef.Value
				}
				return nil
			}
			return nil
		} else {
			return nil
		}
	}

	return current
}

// simplifySchema simplifies a JSON Schema definition for MCP tool input.
func (g *Generator) simplifySchema(schema *openapi3.Schema) map[string]interface{} {
	if schema == nil {
		return map[string]interface{}{"type": "string"}
	}

	result := make(map[string]interface{})

	// Handle Type - it's *Types, use Is() method
	if schema.Type != nil && !schema.Type.IsEmpty() {
		if schema.Type.IsSingle() {
			result["type"] = schema.Type.Slice()[0]
		} else {
			result["type"] = schema.Type.Slice()
		}
	}

	if schema.Description != "" {
		result["description"] = schema.Description
	}

	if len(schema.Enum) > 0 {
		enumVals := make([]interface{}, len(schema.Enum))
		for i, v := range schema.Enum {
			enumVals[i] = v
		}
		result["enum"] = enumVals
	}

	if schema.Format != "" {
		result["format"] = schema.Format
	}

	if schema.Min != nil {
		result["minimum"] = *schema.Min
	}

	if schema.Max != nil {
		result["maximum"] = *schema.Max
	}

	// MinLength is uint64, MaxLength is *uint64
	if schema.MinLength > 0 {
		result["minLength"] = schema.MinLength
	}
	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}

	if schema.Pattern != "" {
		result["pattern"] = schema.Pattern
	}

	if schema.Default != nil {
		result["default"] = schema.Default
	}

	if schema.Nullable {
		result["nullable"] = true
	}

	if schema.Example != nil {
		result["example"] = schema.Example
	}

	if len(schema.AllOf) > 0 {
		// Use first schema in AllOf that has properties
		for _, subSchemaRef := range schema.AllOf {
			if subSchemaRef.Value != nil && subSchemaRef.Value.Properties != nil {
				subResult := g.simplifySchema(subSchemaRef.Value)
				for k, v := range subResult {
					result[k] = v
				}
				break
			}
		}
	}

	if schema.Properties != nil {
		result["properties"] = g.simplifyObjectSchema(schema, nil)
		if schema.Required != nil {
			result["required"] = schema.Required
		}
	}

	if schema.Items != nil && schema.Items.Value != nil {
		result["items"] = g.simplifySchema(schema.Items.Value)
	}

	if len(result) == 0 {
		return map[string]interface{}{"type": "string"}
	}

	return result
}

// simplifyObjectSchema simplifies an object schema with nested properties.
func (g *Generator) simplifyObjectSchema(schema *openapi3.Schema, spec *openapi3.T) map[string]interface{} {
	result := make(map[string]interface{})
	properties := make(map[string]interface{})
	var required []string

	if schema.Properties != nil {
		for propName, propSchemaRef := range schema.Properties {
			var propSchema *openapi3.Schema
			if propSchemaRef.Value != nil {
				propSchema = propSchemaRef.Value
			} else if propSchemaRef.Ref != "" {
				propSchema = g.resolveSchemaRef(propSchemaRef.Ref, spec)
			}
			if propSchema != nil {
				properties[propName] = g.simplifySchema(propSchema)
			}
		}
	}

	for _, req := range schema.Required {
		required = append(required, req)
	}

	if len(properties) > 0 {
		result["properties"] = properties
	}
	if len(required) > 0 {
		result["required"] = required
	}

	return result
}

// buildInputSchema constructs the inputSchema for a tool.
func (g *Generator) buildInputSchema(properties map[string]interface{}, required []string) map[string]interface{} {
	inputSchema := map[string]interface{}{"type": "object"}

	if len(properties) > 0 {
		inputSchema["properties"] = properties
	}

	if len(required) > 0 {
		// Dedupe required fields
		seen := make(map[string]bool)
		var uniqueRequired []string
		for _, r := range required {
			if !seen[r] {
				seen[r] = true
				uniqueRequired = append(uniqueRequired, r)
			}
		}
		inputSchema["required"] = uniqueRequired
	}

	return inputSchema
}

// truncateDescription truncates a description to maxLen at sentence boundary.
func (g *Generator) truncateDescription(description string, maxLen int) string {
	if utf8.RuneCountInString(description) <= maxLen {
		return description
	}

	// Truncate to maxLen first
	utf8Desc := []rune(description)
	if len(utf8Desc) > maxLen {
		utf8Desc = utf8Desc[:maxLen]
	}
	truncated := string(utf8Desc)

	// Find last sentence boundary
	sentenceEnd := g.lastIndexOfAny(truncated, ". ", "? ", "! ", "\n\n")

	// If found after position 700, cut there; otherwise use ellipsis
	if sentenceEnd > 700 {
		return string([]rune(description)[:sentenceEnd+1])
	}

	// Use ellipsis but cap at actual string length
	maxSlice := 897
	if len([]rune(description)) < maxSlice {
		maxSlice = len([]rune(description))
	}
	return string([]rune(description)[:maxSlice]) + "..."
}

// lastIndexOfAny finds the last occurrence of any of the needles in haystack.
func (g *Generator) lastIndexOfAny(haystack string, needles ...string) int {
	lastIdx := -1
	for _, needle := range needles {
		idx := strings.LastIndex(haystack, needle)
		if idx > lastIdx {
			lastIdx = idx
		}
	}
	return lastIdx
}

// isDestructiveOperation determines if an operation is state-mutating/destructive.
func (g *Generator) isDestructiveOperation(httpMethod, path, operationId string) bool {
	// DELETE is always destructive
	if httpMethod == "DELETE" {
		return true
	}

	lowerPath := strings.ToLower(path)

	// Paths containing admin orders are destructive
	if strings.Contains(lowerPath, "/admin/orders/") {
		return true
	}

	// For GET/POST/PUT/PATCH, check destructive path terms
	method := strings.ToUpper(httpMethod)
	if method == "GET" {
		// GET is destructive only for certain sensitive operations
		destructiveGETTerms := []string{
			"reset_password", "change_password", "change_root_password",
			"reinstall", "restore", "ipmi_power", "powerstrip",
		}
		for _, term := range destructiveGETTerms {
			if strings.Contains(lowerPath, term) {
				return true
			}
		}
		return false
	}

	// For POST/PUT/PATCH, check all destructive terms
	destructivePathTerms := []string{
		"cancel", "delete", "refund", "purge", "wipe", "remove",
		"destroy", "reinstall", "reset_password", "change_root_password",
		"change_password", "mark_fraud", "disable", "suspend",
		"restore", "change_ip", "migration", "ipmi_power", "powerstrip",
		"null_routes", "clean_login_logs", "switch_port", "switchport_config",
		"mass_email", "buy_hd_space", "buy_ip",
	}

	for _, term := range destructivePathTerms {
		if strings.Contains(lowerPath, term) {
			if method == "POST" || method == "PUT" || method == "PATCH" {
				return true
			}
		}
	}

	// Check operationId patterns
	if operationId != "" {
		destructiveIdPatterns := regexp.MustCompile(`(?i)^admin(Cancel|Delete|Refund|Reassign|Suspend|Wipe|Purge|Remove|ResetPassword|ResetMailPassword|ReinstallOs|MarkFraud|Destroy|Restore|Migrate|MassEmail|ApcPower|Apc(Setup|Powerstrip)|IpmiPower|ChangeIp|ChangePassword|ChangeRootPassword|CleanLoginLogs|Order|ManageSwitchPort|AddNullRoute|ServerIpmiPower|BuyHdSpace|BuyIp|ForceDelete)`)
		if destructiveIdPatterns.MatchString(operationId) {
			return true
		}
	}

	return false
}

// isIdempotent returns true for HTTP methods that are idempotent.
func (g *Generator) isIdempotent(httpMethod string) bool {
	switch httpMethod {
	case "GET", "PUT", "DELETE":
		return true
	default:
		return false
	}
}

// generateOperationId generates an operationId from path and method if not provided.
func (g *Generator) generateOperationId(path, method string) string {
	// Split path and filter empty parts
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var nameParts []string
	nameParts = append(nameParts, strings.ToLower(method))

	for _, part := range parts {
		// Skip path parameters
		if strings.HasPrefix(part, "{") {
			continue
		}
		// Replace non-alphanumeric with underscore
		sanitized := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(part, "_")
		if sanitized != "" {
			nameParts = append(nameParts, sanitized)
		}
	}

	return strings.Join(nameParts, "_")
}
