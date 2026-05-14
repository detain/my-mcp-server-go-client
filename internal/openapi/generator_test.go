package openapi

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestNewGenerator(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
}

func TestGenerator_ExtractTools_EmptySpec(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	tests := []struct {
		name string
		spec *openapi3.T
	}{
		{"nil spec", nil},
		{"nil paths", &openapi3.T{Paths: nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := gen.ExtractTools(tt.spec)
			if tools == nil {
				t.Error("Expected empty slice, got nil")
			}
		})
	}

	// Test empty paths - need to use NewPaths
	t.Run("empty paths", func(t *testing.T) {
		spec := &openapi3.T{
			OpenAPI: "3.0.0",
			Info: &openapi3.Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Paths: openapi3.NewPaths(),
		}
		tools := gen.ExtractTools(spec)
		if tools == nil {
			t.Error("Expected empty slice, got nil")
		}
	})
}

func TestGenerator_ExtractTools_SinglePath(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	// Create paths and add a path item
	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUsers",
			Summary:     "Get all users",
			Description: "Retrieves a list of all users in the system",
			Tags:        []string{"users"},
		},
	})

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}

	tools := gen.ExtractTools(spec)

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != "getUsers" {
		t.Errorf("Expected name 'getUsers', got %q", tool.Name)
	}
	if tool.HTTPMethod != "GET" {
		t.Errorf("Expected HTTPMethod 'GET', got %q", tool.HTTPMethod)
	}
	if tool.Path != "/users" {
		t.Errorf("Expected path '/users', got %q", tool.Path)
	}
}

func TestGenerator_ExtractTools_MultipleOperations(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUsers",
			Summary:     "Get all users",
		},
		Post: &openapi3.Operation{
			OperationID: "createUser",
			Summary:     "Create a user",
		},
		Delete: &openapi3.Operation{
			OperationID: "deleteUsers",
			Summary:     "Delete all users",
		},
	})
	paths.Set("/users/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUserById",
			Summary:     "Get user by ID",
		},
		Put: &openapi3.Operation{
			OperationID: "updateUser",
			Summary:     "Update a user",
		},
		Delete: &openapi3.Operation{
			OperationID: "deleteUser",
			Summary:     "Delete a user",
		},
	})

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}

	tools := gen.ExtractTools(spec)

	if len(tools) != 6 {
		t.Errorf("Expected 6 tools, got %d", len(tools))
	}
}

func TestGenerator_ExtractTools_WithPathParams(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	paths := openapi3.NewPaths()
	paths.Set("/users/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUserById",
			Summary:     "Get user by ID",
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{
						Name:        "id",
						In:          "path",
						Required:    true,
						Description: "User ID",
						Schema:      &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
					},
				},
			},
		},
	})

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}

	tools := gen.ExtractTools(spec)

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if len(tool.PathParams) != 1 {
		t.Errorf("Expected 1 path param, got %d", len(tool.PathParams))
	}
	if tool.PathParams[0] != "id" {
		t.Errorf("Expected path param 'id', got %q", tool.PathParams[0])
	}
	if tool.InputSchema["required"] == nil {
		t.Error("Expected 'id' to be required")
	}
}

func TestGenerator_ExtractTools_WithQueryParams(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUsers",
			Summary:     "Get all users",
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{
						Name:     "limit",
						In:       "query",
						Required: false,
						Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
					},
				},
				{
					Value: &openapi3.Parameter{
						Name:     "offset",
						In:       "query",
						Required: false,
						Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
					},
				},
			},
		},
	})

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}

	tools := gen.ExtractTools(spec)

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if len(tool.QueryParams) != 2 {
		t.Errorf("Expected 2 query params, got %d", len(tool.QueryParams))
	}
}

func TestGenerator_ExtractTools_WithTags(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUsers",
			Summary:     "Get all users",
			Tags:        []string{"users"},
		},
	})

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}

	tools := gen.ExtractTools(spec)

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if !strings.Contains(tool.Description, "[users]") {
		t.Errorf("Expected description to contain [users] tag, got %q", tool.Description)
	}
}

func TestGenerator_ExtractTools_DestructiveOperations(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	testCases := []struct {
		name            string
		path            string
		method          string
		operation       *openapi3.Operation
		expectDestructive bool
	}{
		{
			"DELETE method",
			"/users/{id}",
			"DELETE",
			&openapi3.Operation{
				OperationID: "deleteUser",
				Summary:     "Delete",
			},
			true,
		},
		{
			"cancel in path",
			"/orders/{id}/cancel",
			"POST",
			&openapi3.Operation{
				OperationID: "cancelOrder",
				Summary:     "Cancel order",
			},
			true,
		},
		{
			"adminCancel in operationId",
			"/orders",
			"POST",
			&openapi3.Operation{
				OperationID: "adminCancelOrder",
				Summary:     "Admin cancel",
			},
			true,
		},
		{
			"normal GET",
			"/users",
			"GET",
			&openapi3.Operation{
				OperationID: "getUsers",
				Summary:     "Get",
			},
			false,
		},
		{
			"normal POST",
			"/users",
			"POST",
			&openapi3.Operation{
				OperationID: "createUser",
				Summary:     "Create",
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paths := openapi3.NewPaths()
			paths.Set(tc.path, &openapi3.PathItem{})

			// Set the operation on the path item
			pathItem := paths.Value(tc.path)
			switch tc.method {
			case "GET":
				pathItem.Get = tc.operation
			case "POST":
				pathItem.Post = tc.operation
			case "DELETE":
				pathItem.Delete = tc.operation
			case "PUT":
				pathItem.Put = tc.operation
			}

			spec := &openapi3.T{
				OpenAPI: "3.0.0",
				Info: &openapi3.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: paths,
			}

			tools := gen.ExtractTools(spec)

			if len(tools) != 1 {
				t.Fatalf("Expected 1 tool, got %d", len(tools))
			}

			tool := tools[0]
			if tc.expectDestructive {
				if !strings.Contains(tool.Description, "[DESTRUCTIVE]") {
					t.Errorf("Expected [DESTRUCTIVE] in description, got %q", tool.Description)
				}
				if !tool.Annotations.DestructiveHint {
					t.Error("Expected DestructiveHint to be true")
				}
			} else {
				if strings.Contains(tool.Description, "[DESTRUCTIVE]") {
					t.Errorf("Expected no [DESTRUCTIVE] in description, got %q", tool.Description)
				}
			}
		})
	}
}

func TestGenerator_ExtractTools_WithRequestBody(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "createUser",
			Summary:     "Create a user",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: openapi3.Content{
						"application/json": {
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"object"},
									Properties: openapi3.Schemas{
										"name":  {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
										"email": {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
									},
									Required: []string{"name", "email"},
								},
							},
						},
					},
				},
			},
		},
	})

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}

	tools := gen.ExtractTools(spec)

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if !tool.HasBody {
		t.Error("Expected HasBody to be true")
	}

	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties in inputSchema")
	}

	if props["name"] == nil {
		t.Error("Expected 'name' property")
	}
	if props["email"] == nil {
		t.Error("Expected 'email' property")
	}
}

func TestGenerator_ExtractTools_Annotations(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUsers",
			Summary:     "Get all users",
		},
		Delete: &openapi3.Operation{
			OperationID: "deleteUsers",
			Summary:     "Delete all users",
		},
		Put: &openapi3.Operation{
			OperationID: "replaceUsers",
			Summary:     "Replace all users",
		},
	})

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: paths,
	}

	tools := gen.ExtractTools(spec)

	expected := map[string]struct {
		readOnly    bool
		destructive bool
		idempotent  bool
	}{
		"getUsers":     {readOnly: true, destructive: false, idempotent: true},
		"deleteUsers":  {readOnly: false, destructive: true, idempotent: true},
		"replaceUsers": {readOnly: false, destructive: false, idempotent: true},
	}

	for _, tool := range tools {
		exp, ok := expected[tool.Name]
		if !ok {
			continue
		}

		if tool.Annotations.ReadOnlyHint != exp.readOnly {
			t.Errorf("Tool %s: expected ReadOnlyHint=%v, got %v", tool.Name, exp.readOnly, tool.Annotations.ReadOnlyHint)
		}
		if tool.Annotations.DestructiveHint != exp.destructive {
			t.Errorf("Tool %s: expected DestructiveHint=%v, got %v", tool.Name, exp.destructive, tool.Annotations.DestructiveHint)
		}
		if tool.Annotations.IdempotentHint != exp.idempotent {
			t.Errorf("Tool %s: expected IdempotentHint=%v, got %v", tool.Name, exp.idempotent, tool.Annotations.IdempotentHint)
		}
		if !tool.Annotations.OpenWorldHint {
			t.Errorf("Tool %s: expected OpenWorldHint=true", tool.Name)
		}
	}
}

func TestGenerator_truncateDescription(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	tests := []struct {
		name           string
		input          string
		maxLen         int
		expectEllipsis bool
	}{
		{
			name:           "short string",
			input:          "This is a short description.",
			maxLen:         900,
			expectEllipsis: false,
		},
		{
			name:           "exactly max len",
			input:          strings.Repeat("a", 900),
			maxLen:         900,
			expectEllipsis: false,
		},
		{
			name:           "longer than 900 chars with sentence boundary",
			input:          "This is a test. " + strings.Repeat("word ", 200),
			maxLen:         900,
			expectEllipsis: false,
		},
		{
			name:           "longer than 900 chars without good sentence boundary",
			input:          strings.Repeat("a", 1000),
			maxLen:         900,
			expectEllipsis: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.truncateDescription(tt.input, tt.maxLen)
			runeCount := 0
			for _, r := range result {
				_ = r
				runeCount++
			}
			// Truncation is to ~900 chars, not maxLen
			if tt.maxLen == 900 && runeCount > 905 {
				t.Errorf("Result too long: %d runes (expected ~900)", runeCount)
			}
		})
	}
}

func TestGenerator_generateOperationId(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	tests := []struct {
		path     string
		method   string
		expected string
	}{
		{"/users", "get", "get_users"},
		{"/users/{id}", "get", "get_users"},
		{"/users/{id}/posts", "post", "post_users_posts"},
		{"/api/v2/admin/users", "delete", "delete_api_v2_admin_users"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := gen.generateOperationId(tt.path, tt.method)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerator_isDestructiveOperation(t *testing.T) {
	logger := slog.Default()
	gen := NewGenerator(logger)

	tests := []struct {
		method      string
		path        string
		operationId string
		expectDest  bool
	}{
		{"DELETE", "/users", "", true},
		{"DELETE", "/users/{id}", "", true},
		{"GET", "/users", "", false},
		{"POST", "/orders/{id}/cancel", "", true},
		{"PUT", "/users/{id}/disable", "", true},
		{"GET", "/users/{id}/reset_password", "", true},
		{"GET", "/users/{id}/change_password", "", true},
		{"POST", "/orders", "", false},
		{"POST", "", "adminCancelOrder", true},
		{"POST", "", "createUser", false},
		{"DELETE", "", "adminDeleteSomething", true},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path+" "+tt.operationId, func(t *testing.T) {
			result := gen.isDestructiveOperation(tt.method, tt.path, tt.operationId)
			if result != tt.expectDest {
				t.Errorf("isDestructiveOperation(%q, %q, %q) = %v, expected %v",
					tt.method, tt.path, tt.operationId, result, tt.expectDest)
			}
		})
	}
}
