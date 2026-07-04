package schema

import (
	"encoding/json"
	"testing"
)

func TestNewValidator(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`)

	validator, err := NewValidator(schemaJSON)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}
	if validator == nil {
		t.Fatal("validator is nil")
	}
}

func TestNewValidatorInvalidJSON(t *testing.T) {
	schemaJSON := []byte(`{invalid json}`)
	_, err := NewValidator(schemaJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateValidObject(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`)

	validator, _ := NewValidator(schemaJSON)

	data := map[string]interface{}{
		"name": "John",
		"age":  30.0,
	}

	result := validator.Validate(data)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateMissingRequired(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`)

	validator, _ := NewValidator(schemaJSON)

	data := map[string]interface{}{
		"age": 30.0,
	}

	result := validator.Validate(data)
	if result.Valid {
		t.Error("expected invalid for missing required field")
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestValidateTypeMismatch(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`)

	validator, _ := NewValidator(schemaJSON)

	data := map[string]interface{}{
		"name": 123,
	}

	result := validator.Validate(data)
	if result.Valid {
		t.Error("expected invalid for type mismatch")
	}
}

func TestValidateEnum(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"color": {"type": "string", "enum": ["red", "green", "blue"]}
		}
	}`)

	validator, _ := NewValidator(schemaJSON)

	// Valid value
	result := validator.Validate(map[string]interface{}{
		"color": "red",
	})
	if !result.Valid {
		t.Error("expected valid for enum match")
	}

	// Invalid value
	result = validator.Validate(map[string]interface{}{
		"color": "yellow",
	})
	if result.Valid {
		t.Error("expected invalid for enum mismatch")
	}
}

func TestValidateNumberConstraints(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"score": {"type": "number", "minimum": 0, "maximum": 100}
		}
	}`)

	validator, _ := NewValidator(schemaJSON)

	// Valid value
	result := validator.Validate(map[string]interface{}{
		"score": 50.0,
	})
	if !result.Valid {
		t.Error("expected valid for in-range value")
	}

	// Too low
	result = validator.Validate(map[string]interface{}{
		"score": -1.0,
	})
	if result.Valid {
		t.Error("expected invalid for value below minimum")
	}

	// Too high
	result = validator.Validate(map[string]interface{}{
		"score": 101.0,
	})
	if result.Valid {
		t.Error("expected invalid for value above maximum")
	}
}

func TestValidateArray(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"}
			}
		}
	}`)

	validator, _ := NewValidator(schemaJSON)

	// Valid array
	result := validator.Validate(map[string]interface{}{
		"tags": []interface{}{"go", "mcp"},
	})
	if !result.Valid {
		t.Error("expected valid array")
	}

	// Invalid array item
	result = validator.Validate(map[string]interface{}{
		"tags": []interface{}{"go", 123},
	})
	if result.Valid {
		t.Error("expected invalid array item")
	}
}

func TestExtractSchemaFromJSON(t *testing.T) {
	data := []byte(`{
		"name": "test",
		"count": 42,
		"active": true,
		"tags": ["a", "b"]
	}`)

	schema, err := ExtractSchemaFromJSON(data)
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("expected type object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties")
	}

	if props["name"] == nil {
		t.Error("expected name property")
	}
	if props["count"] == nil {
		t.Error("expected count property")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"id": {"type": "number"},
			"name": {"type": "string"}
		}
	}`)

	validator, _ := NewValidator(schemaJSON)

	data := map[string]interface{}{
		"id":   1.0,
		"name": "test",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var roundTripped map[string]interface{}
	if err := json.Unmarshal(jsonData, &roundTripped); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Validate round-tripped data
	result := validator.Validate(roundTripped)
	if !result.Valid {
		t.Errorf("expected valid after round-trip, got errors: %v", result.Errors)
	}
}
