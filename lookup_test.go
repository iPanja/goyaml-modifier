package modifier

import (
	"reflect"
	"strings"
	"testing"
)

// TestBuildMapLookup tests the buildMapLookup function
func TestBuildMapLookup(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
		skip     map[string]bool // fields that should be skipped
	}{
		{
			name: "simple string map",
			input: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "map with skipped fields",
			input: map[string]interface{}{
				"valid":   "value",
				"invalid": nil,
				"empty":   "",
			},
			expected: map[string]interface{}{
				"valid": "value",
				"empty": "",
			},
			skip: map[string]bool{
				"invalid": true,
			},
		},
		{
			name: "nested map structure",
			input: map[string]interface{}{
				"top": map[string]interface{}{
					"nested": "value",
				},
				"other": 123,
			},
			expected: map[string]interface{}{
				"top": map[string]interface{}{
					"nested": "value",
				},
				"other": 123,
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	// Mock ShouldSkipField function for testing
	originalShouldSkipField := ShouldSkipField
	defer func() { ShouldSkipField = originalShouldSkipField }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock for ShouldSkipField
			ShouldSkipField = func(v reflect.Value) bool {
				if tt.skip == nil {
					return originalShouldSkipField(v)
				}
				if v.Kind() == reflect.Interface && v.IsNil() {
					return true
				}
				if v.Kind() == reflect.String && v.String() == "" {
					return false // don't skip empty strings in this test
				}
				return false
			}

			inputVal := reflect.ValueOf(tt.input)
			lookup := make(map[string]reflect.Value)
			buildMapLookup(inputVal, lookup)

			// Verify the results
			if len(lookup) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(lookup))
			}

			for key, expectedVal := range tt.expected {
				actualVal, ok := lookup[key]
				if !ok {
					t.Errorf("missing expected key: %s", key)
					continue
				}

				if !reflect.DeepEqual(actualVal.Interface(), expectedVal) {
					t.Errorf("for key %s: expected %v (%T), got %v (%T)",
						key, expectedVal, expectedVal, actualVal.Interface(), actualVal.Interface())
				}
			}

			// Check for unexpected keys
			for key := range lookup {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("unexpected key in result: %s", key)
				}
			}
		})
	}
}

// TestBuildStructLookup tests the buildStructLookup function
func TestBuildStructLookup(t *testing.T) {
	// Mock functions
	originalIsInline := IsInlineStructField
	originalShouldSkip := ShouldSkipField
	originalIsOmitEmpty := IsOmitEmptyStructField
	originalGetKey := getStructFieldKey

	defer func() {
		IsInlineStructField = originalIsInline
		ShouldSkipField = originalShouldSkip
		IsOmitEmptyStructField = originalIsOmitEmpty
		getStructFieldKey = originalGetKey
	}()

	tests := []struct {
		name           string
		input          interface{}
		mockIsInline   func(reflect.StructField) bool
		mockShouldSkip func(reflect.Value) bool
		mockOmitEmpty  func(reflect.StructField) bool
		mockGetKey     func(reflect.StructField) string
		expected       map[string]interface{}
	}{
		{
			name: "simple struct",
			input: struct {
				Name string `yaml:"name"`
				Age  int    `yaml:"age"`
			}{
				Name: "Alice",
				Age:  30,
			},
			mockGetKey: func(f reflect.StructField) string {
				return f.Tag.Get("yaml")
			},
			expected: map[string]interface{}{
				"name": "Alice",
				"age":  30,
			},
		},
		{
			name: "with skipped field",
			input: struct {
				Name string `yaml:"name"`
				Skip string `yaml:"-"`
			}{
				Name: "Bob",
				Skip: "should-be-skipped",
			},
			mockShouldSkip: func(v reflect.Value) bool {
				return v.String() == "should-be-skipped"
			},
			mockGetKey: func(f reflect.StructField) string {
				if f.Tag.Get("yaml") == "-" {
					return ""
				}
				return f.Tag.Get("yaml")
			},
			expected: map[string]interface{}{
				"name": "Bob",
			},
		},
		{
			name: "with omitempty",
			input: struct {
				Name string `yaml:"name,omitempty"`
				Age  int    `yaml:"age,omitempty"`
			}{
				Name: "",
				Age:  0,
			},
			mockOmitEmpty: func(f reflect.StructField) bool {
				tag := f.Tag.Get("yaml")
				return tag == "name,omitempty" || tag == "age,omitempty"
			},
			mockGetKey: func(f reflect.StructField) string {
				return f.Tag.Get("yaml")
			},
			expected: map[string]interface{}{},
		},
		{
			name: "with inline struct",
			input: struct {
				Name    string `yaml:"name"`
				Address struct {
					City string `yaml:"city"`
				} `yaml:",inline"`
			}{
				Name: "Charlie",
				Address: struct {
					City string `yaml:"city"`
				}{
					City: "London",
				},
			},
			mockIsInline: func(f reflect.StructField) bool {
				return f.Tag.Get("yaml") == ",inline"
			},
			mockGetKey: func(f reflect.StructField) string {
				if f.Tag.Get("yaml") == ",inline" {
					return ""
				}
				return f.Tag.Get("yaml")
			},
			expected: map[string]interface{}{
				"name": "Charlie",
				"city": "London",
			},
		},
		{
			name: "unexported field",
			input: struct {
				name string // unexported
				Age  int    `yaml:"age"`
			}{
				name: "hidden",
				Age:  25,
			},
			expected: map[string]interface{}{
				"age": 25,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			if tt.mockIsInline != nil {
				IsInlineStructField = tt.mockIsInline
			} else {
				IsInlineStructField = func(f reflect.StructField) bool { return false }
			}

			if tt.mockShouldSkip != nil {
				ShouldSkipField = tt.mockShouldSkip
			} else {
				ShouldSkipField = func(v reflect.Value) bool { return false }
			}

			if tt.mockOmitEmpty != nil {
				IsOmitEmptyStructField = tt.mockOmitEmpty
			} else {
				IsOmitEmptyStructField = func(f reflect.StructField) bool { return false }
			}

			if tt.mockGetKey != nil {
				getStructFieldKey = tt.mockGetKey
			} else {
				getStructFieldKey = func(f reflect.StructField) string { return strings.ToLower(f.Name) }
			}

			lookup := make(map[string]reflect.Value)
			buildStructLookup(reflect.ValueOf(tt.input), lookup)

			// Verify results
			if len(lookup) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(lookup))
			}

			for key, expectedVal := range tt.expected {
				actualVal, ok := lookup[key]
				if !ok {
					t.Errorf("missing expected key: %s", key)
					continue
				}

				if !reflect.DeepEqual(actualVal.Interface(), expectedVal) {
					t.Errorf("for key %s: expected %v (%T), got %v (%T)",
						key, expectedVal, expectedVal, actualVal.Interface(), actualVal.Interface())
				}
			}

			// Check for unexpected keys
			for key := range lookup {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("unexpected key in result: %s", key)
				}
			}
		})
	}
}
