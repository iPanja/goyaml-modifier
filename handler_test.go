package modifier

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHandlerFull(t *testing.T) {
	var tests = []struct {
		name      string
		input     string
		newData   func() any
		isFocused bool
	}{
		{
			name:  "Extensive test via maps",
			input: "everything.yaml",
			newData: func() any {
				return map[string]any{
					"data": map[string]any{
						"string":  "MODIFIED STRING",
						"integer": 999,
						"boolean": false,
						"quoted_strings": map[string]any{
							"single_quoted": "MODIFIED SINGLE QUOTED",
							"double_quoted": "MODIFIED DOUBLE QUOTED",
							// "multiline":     "This is a multiline string\nthat preserves newlines",
							// "folded":        "This is a folded string where newlines become spaces unless they're empty lines or more indented.",
						},
						"collections": map[string]any{
							"simple_list": []int{9, 8, 7, 6},
							"mixed_list":  []any{1, "two", 3, true},
							"nested_list": [][]int{
								{10, 20},
								{30, 40},
							},
							"simple_map": map[string]any{
								"a": 1,
								"b": 2,
								"x": 100,
							},
							"mixed_map": map[string]any{
								"a": 1,
								"b": "two",
								"c": 3,
							},
							"list_of_maps": []any{
								map[string]any{
									"name": "MODIFIED ALICE",
									"age":  99,
								},
								map[string]any{
									"name": "Bob",
									"age":  25,
								},
							},
						},
						"anchors_aliases": map[string]any{
							"base": map[string]any{
								"name":  "MODIFIED BASE",
								"value": 100,
							},
							"first": map[string]any{
								"extra": "MODIFIED FIRST",
							},
							"second": map[string]any{
								"override": 999,
							},
						},
						"deeply_nested": map[string]any{
							"level1": map[string]any{
								"level2": map[string]any{
									"level3": map[string]any{
										"level4": map[string]any{
											"value": "MODIFIED DEEP VALUE",
										},
										"list": []int{1, 2, 3},
										"another_level3": []string{
											"item1",
											"item2",
											"new_item",
										},
									},
								},
								"scalar_at_depth": "value",
							},
						},
						"document_two": map[string]any{
							"description": "This is a separate YAML document",
							"items": []any{
								"first",
								"second",
								"third",
							},
						},
					},
				}
			},
		},
	}

	areAnyFocused := false
	for _, test := range tests {
		if test.isFocused {
			areAnyFocused = true
			break
		}
	}

	for _, tt := range tests {
		if !tt.isFocused && areAnyFocused {
			continue
		}

		t.Run(tt.name, func(t *testing.T) {
			// Load the YAML file
			inputPath := filepath.Join("testdata", tt.input)
			expectedPath := filepath.Join("testdata", fmt.Sprintf("expect_%s", tt.input))

			input, _ := os.ReadFile(inputPath)
			expected, _ := os.ReadFile(expectedPath)

			// Process input
			var inNode yaml.Node
			if err := yaml.Unmarshal(input, &inNode); err != nil {
				t.Fatalf("failed to unmarshal input: %v", err)
			}

			// Load into map for debugging
			var inMap map[string]any
			if err := inNode.Decode(&inMap); err != nil {
				t.Fatalf("failed to decode input: %v", err)
			}

			// Create a new YAMLHandler
			c := inNode.Content[0]
			handler := NewYAMLHandler(c)
			handler.overrideExplicit = true
			handler.Update(tt.newData())

			// Marshal the output
			out, err := yaml.Marshal(handler.in)
			if err != nil {
				t.Fatalf("failed to marshal output: %v", err)
			}
			out = removeMergeTags(out)

			// Write the output to a file (for debugging purposes)
			if tt.isFocused {
				outputPath := fmt.Sprintf("out_%s", tt.input)
				os.WriteFile(outputPath, out, 0644)
			}

			// Compare the output with the expected output
			if string(out) != string(expected) {
				t.Errorf("output does not match expected output\nExpected:\n%s\nGot:\n%s", expected, out)
			}
		})
	}
}

func removeMergeTags(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("!!merge "), []byte(""))
}
