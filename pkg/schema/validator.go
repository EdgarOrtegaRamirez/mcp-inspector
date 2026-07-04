package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Validator validates data against JSON Schema
type Validator struct {
	schema map[string]interface{}
}

// NewValidator creates a new validator from a JSON Schema
func NewValidator(schemaJSON []byte) (*Validator, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}
	return &Validator{schema: schema}, nil
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Validate validates data against the schema
func (v *Validator) Validate(data interface{}) *ValidationResult {
	errors := []ValidationError{}
	v.validateNode(v.schema, data, "", &errors)
	return &ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

func (v *Validator) validateNode(schema map[string]interface{}, data interface{}, path string, errors *[]ValidationError) {
	// Check type
	if typ, ok := schema["type"]; ok {
		typeName, _ := typ.(string)
		actualType := getJSONType(data)
		if typeName != "any" && typeName != actualType {
			*errors = append(*errors, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("expected type %s, got %s", typeName, actualType),
			})
			return
		}
	}

	// Check required fields
	if required, ok := schema["required"]; ok {
		if reqList, ok := required.([]interface{}); ok {
			if obj, ok := data.(map[string]interface{}); ok {
				for _, req := range reqList {
					if fieldName, ok := req.(string); ok {
						if _, exists := obj[fieldName]; !exists {
							*errors = append(*errors, ValidationError{
								Path:    path + "." + fieldName,
								Message: fmt.Sprintf("required field %q is missing", fieldName),
							})
						}
					}
				}
			}
		}
	}

	// Check enum
	if enum, ok := schema["enum"]; ok {
		if enumList, ok := enum.([]interface{}); ok {
			found := false
			for _, e := range enumList {
				if fmt.Sprintf("%v", e) == fmt.Sprintf("%v", data) {
					found = true
					break
				}
			}
			if !found {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("value not in enum: %v", enumList),
				})
			}
		}
	}

	// Check minimum/maximum for numbers
	if num, ok := data.(float64); ok {
		if min, ok := schema["minimum"]; ok {
			if minVal, ok := min.(float64); ok {
				if num < minVal {
					*errors = append(*errors, ValidationError{
						Path:    path,
						Message: fmt.Sprintf("value %v is less than minimum %v", num, minVal),
					})
				}
			}
		}
		if max, ok := schema["maximum"]; ok {
			if maxVal, ok := max.(float64); ok {
				if num > maxVal {
					*errors = append(*errors, ValidationError{
						Path:    path,
						Message: fmt.Sprintf("value %v is greater than maximum %v", num, maxVal),
					})
				}
			}
		}
	}

	// Check string constraints
	if str, ok := data.(string); ok {
		if minLen, ok := schema["minLength"]; ok {
			if ml, ok := minLen.(float64); ok {
				if len(str) < int(ml) {
					*errors = append(*errors, ValidationError{
						Path:    path,
						Message: fmt.Sprintf("string length %d is less than minLength %d", len(str), int(ml)),
					})
				}
			}
		}
		if maxLen, ok := schema["maxLength"]; ok {
			if ml, ok := maxLen.(float64); ok {
				if len(str) > int(ml) {
					*errors = append(*errors, ValidationError{
						Path:    path,
						Message: fmt.Sprintf("string length %d is greater than maxLength %d", len(str), int(ml)),
					})
				}
			}
		}
		if pattern, ok := schema["pattern"]; ok {
			if patStr, ok := pattern.(string); ok {
				if matched, _ := matchPattern(patStr, str); !matched {
					*errors = append(*errors, ValidationError{
						Path:    path,
						Message: fmt.Sprintf("string does not match pattern %q", patStr),
					})
				}
			}
		}
	}

	// Check properties for objects
	if properties, ok := schema["properties"]; ok {
		if propMap, ok := properties.(map[string]interface{}); ok {
			if obj, ok := data.(map[string]interface{}); ok {
				for propName, propSchema := range propMap {
					if propSchemaMap, ok := propSchema.(map[string]interface{}); ok {
						if val, exists := obj[propName]; exists {
							newPath := path
							if newPath != "" {
								newPath += "."
							}
							newPath += propName
							v.validateNode(propSchemaMap, val, newPath, errors)
						}
					}
				}
			}
		}
	}

	// Check items for arrays
	if items, ok := schema["items"]; ok {
		if itemSchema, ok := items.(map[string]interface{}); ok {
			if arr, ok := data.([]interface{}); ok {
				for i, item := range arr {
					newPath := fmt.Sprintf("%s[%d]", path, i)
					v.validateNode(itemSchema, item, newPath, errors)
				}
			}
		}
	}
}

func getJSONType(data interface{}) string {
	switch data.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

func matchPattern(pattern, s string) (bool, error) {
	// Simple pattern matching for common cases
	// In production, use regexp package
	if strings.HasPrefix(pattern, "^") && strings.HasSuffix(pattern, "$") {
		inner := strings.TrimPrefix(pattern, "^")
		inner = strings.TrimSuffix(inner, "$")
		return strings.Contains(s, inner), nil
	}
	return strings.Contains(s, pattern), nil
}

// ExtractSchemaFromJSON infers a JSON Schema from sample JSON data
func ExtractSchemaFromJSON(data []byte) (map[string]interface{}, error) {
	var sample interface{}
	if err := json.Unmarshal(data, &sample); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	schema := inferSchema(sample)
	return schema, nil
}

func inferSchema(data interface{}) map[string]interface{} {
	schema := make(map[string]interface{})

	switch v := data.(type) {
	case nil:
		schema["type"] = "null"
	case bool:
		schema["type"] = "boolean"
	case float64:
		schema["type"] = "number"
	case string:
		schema["type"] = "string"
		// Try to infer format
		if len(v) == 20 && strings.Contains(v, "T") {
			schema["format"] = "date-time"
		} else if strings.Contains(v, "@") {
			schema["format"] = "email"
		} else if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
			schema["format"] = "uri"
		}
	case []interface{}:
		schema["type"] = "array"
		if len(v) > 0 {
			schema["items"] = inferSchema(v[0])
		}
	case map[string]interface{}:
		schema["type"] = "object"
		props := make(map[string]interface{})
		required := []string{}
		for key, val := range v {
			props[key] = inferSchema(val)
			required = append(required, key)
		}
		schema["properties"] = props
		schema["required"] = required
	}

	return schema
}
