package modifier

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestTransferRequest(t *testing.T) {
	var tests = []struct {
		name       string
		out        *yaml.Node
		ins        []*yaml.Node
		onlyUpdate bool
		isValid    func(t *testing.T, out *yaml.Node, in []*yaml.Node)
	}{
		{
			name: "Valid TransferRequest",
			out: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					scalarNode("key1"),
					scalarNode("value1"),
					scalarNode("key2"),
					scalarNode("value2"),
					scalarNode("key3"),
					scalarNode("value3"),
				},
			},
			ins: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("key1"), // Should update by itself
						scalarNode("new value 1"),
						scalarNode("key2"), // Agrees with next source
						scalarNode("new value 2"),
						scalarNode("key3"),
						scalarNode("ALSO should not get set"), // Disagrees with next source
						scalarNode("key4"),
						scalarNode("new value 4"), // Should get added
					},
				},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("key2"),
						scalarNode("new value 2"),
						scalarNode("key3"),
						scalarNode("should not get set"),
					},
				},
			},
			isValid: func(t *testing.T, out *yaml.Node, in []*yaml.Node) {
				expectedOut := []string{
					"key1",
					"new value 1",
					"key2",
					"new value 2",
					"key3",
					"value3",
					"key4",
					"new value 4",
				}
				expectedIn1 := []string{
					"key3",
					"ALSO should not get set",
				}
				expectedIn2 := []string{
					"key3",
					"should not get set",
				}

				assert.ElementsMatch(t, expectedOut, toArray(out), "TransferRequest should be valid")
				assert.ElementsMatch(t, expectedIn1, toArray(in[0]), "First source should be valid")
				assert.ElementsMatch(t, expectedIn2, toArray(in[1]), "Second source should be valid")
			},
			onlyUpdate: false,
		},
		{
			name: "Valid TransferRequest (update only)",
			out: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					scalarNode("key1"),
					scalarNode("value1"),
					scalarNode("key2"),
					scalarNode("value2"),
					scalarNode("key3"),
					scalarNode("value3"),
				},
			},
			ins: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("key1"), // Should update by itself
						scalarNode("new value 1"),
						scalarNode("key2"), // Agrees with next source
						scalarNode("new value 2"),
						scalarNode("key3"),
						scalarNode("ALSO should not get set"), // Disagrees with next source
						scalarNode("key4"),
						scalarNode("new value 4"), // Should not get added (onlyUpdate == true)
						scalarNode("key5"),
						scalarNode("new value 5"), // Should not get added (onlyUpdate == true)
					},
				},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("key2"),
						scalarNode("new value 2"),
						scalarNode("key3"),
						scalarNode("should not get set"),
						scalarNode("key4"),
						scalarNode("new value 4"), // Should not get added
					},
				},
			},
			isValid: func(t *testing.T, out *yaml.Node, in []*yaml.Node) {
				expectedOut := []string{
					"key1",
					"new value 1",
					"key2",
					"new value 2",
					"key3",
					"value3",
				}
				expectedIn1 := []string{
					"key3",
					"ALSO should not get set",
					"key4",
					"new value 4",
					"key5",
					"new value 5",
				}
				expectedIn2 := []string{
					"key3",
					"should not get set",
					"key4",
					"new value 4",
				}

				assert.ElementsMatch(t, expectedOut, toArray(out), "TransferRequest should be valid")
				assert.ElementsMatch(t, expectedIn1, toArray(in[0]), "First source should be valid")
				assert.ElementsMatch(t, expectedIn2, toArray(in[1]), "Second source should be valid")
			},
			onlyUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TransferRequest{
				ins:        tt.ins,
				out:        tt.out,
				onlyUpdate: tt.onlyUpdate,
			}

			tr.Transfer()
			tt.isValid(t, tr.out, tr.ins)
		})
	}
}

func TestHandleNode(t *testing.T) {
	o := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("key1"),
			scalarNode("value1"),
			scalarNode("key2"),
			scalarNode("value2"),
		},
		Anchor: "anchor1",
	}
	o2 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("key3"),
			scalarNode("value3"),
		},
		Anchor: "anchor2",
	}

	var tests = []struct {
		name    string
		nodes   []*yaml.Node
		isValid func(t *testing.T, trh *TRHandler)
	}{
		{
			name: "Test normal order",
			nodes: []*yaml.Node{
				o,
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("<<"),
						aliasNode(o, "anchor1"),
					},
				},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("<<"),
						aliasNode(o, "anchor1"),
					},
				},
			},
			isValid: func(t *testing.T, trh *TRHandler) {
				assert.Len(t, trh.requests, 1, "Should have 1 request")
				tr := trh.requests[o]
				assert.Len(t, tr.ins, 2, "Should have 2 sources")
				assert.Equal(t, o, tr.out, "Should have the same output node")
			},
		},
		{
			name: "Test without explicitly iterating over the output node",
			nodes: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("<<"),
						aliasNode(o, "anchor1"),
					},
				},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("<<"),
						aliasNode(o, "anchor1"),
					},
				},
			},
			isValid: func(t *testing.T, trh *TRHandler) {
				assert.Len(t, trh.requests, 1, "Should have 1 request")
				tr := trh.requests[o]
				assert.Len(t, tr.ins, 2, "Should have 2 sources")
				assert.Equal(t, o, tr.out, "Should have the same output node")
			},
		},
		{
			name: "Test with multiple output nodes",
			nodes: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("<<"),
						aliasNode(o, "anchor1"),
					},
				},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("<<"),
						aliasNode(o, "anchor1"),
					},
				},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						scalarNode("<<"),
						aliasNode(o2, "anchor2"),
					},
				},
			},
			isValid: func(t *testing.T, trh *TRHandler) {
				assert.Len(t, trh.requests, 2, "Should have 2 request")

				tr1 := trh.requests[o]
				assert.Len(t, tr1.ins, 2, "Should have 2 sources")
				assert.Equal(t, o, tr1.out, "Should have the same output node")

				tr2 := trh.requests[o2]
				assert.Len(t, tr2.ins, 1, "Should have 1 source")
				assert.Equal(t, o2, tr2.out, "Should have the same output node")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trh := TRHandler{
				requests: make(map[*yaml.Node]*TransferRequest),
			}

			for _, node := range tt.nodes {
				trh.HandleNode(node)
			}

			tt.isValid(t, &trh)
		})
	}
}

func TestHandlerSorting(t *testing.T) {
	o2 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("key1"),
			scalarNode("value1"),
			scalarNode("key2"),
			scalarNode("value2"),
		},
		Anchor:      "anchor2",
		LineComment: "o2",
	}
	o1 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("key3"),
			scalarNode("value3"),
			scalarNode("<<"),
			aliasNode(o2, "anchor2"),
		},
		Anchor:      "anchor1",
		LineComment: "o1",
	}
	o3 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("<<"),
			aliasNode(o1, "anchor1"),
			scalarNode("key4"),
			scalarNode("value4"),
		},
		Anchor:      "anchor3",
		LineComment: "o3",
	}

	A := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("<<"),
			aliasNode(o1, "anchor1"),
		},
		LineComment: "A",
	}
	B := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("<<"),
			aliasNode(o1, "anchor1"),
		},
		LineComment: "B",
	}
	C := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("<<"),
			aliasNode(o2, "anchor2"),
		},
		LineComment: "C",
	}

	var tests = []struct {
		nodes     []*yaml.Node
		isFocused bool
	}{
		{
			nodes: []*yaml.Node{o1, o2, C, A, B, o3},
		},
		{
			nodes: []*yaml.Node{o1, C, o2, A, B, o3},
		},
		{
			nodes: []*yaml.Node{o1, A, o2, C, B, o3},
		},
		{
			nodes: []*yaml.Node{o1, A, B, o2, C, o3},
		},
		{
			nodes: []*yaml.Node{o1, A, B, o3, o2, C},
		},
		{
			nodes: []*yaml.Node{o1, A, B, C, o2, o3},
		},
		{
			nodes: []*yaml.Node{o1, A, C, o2, B, o3},
		},
		{
			nodes: []*yaml.Node{o2, o3, o1, A, B, C},
		},
		{
			nodes: []*yaml.Node{o2, o1, A, B, C, o3},
		},
	}

	areAnyFocused := false
	for _, tt := range tests {
		if tt.isFocused {
			areAnyFocused = true
		}
	}

	for _, tt := range tests {
		if areAnyFocused && !tt.isFocused {
			continue
		}

		trh := TRHandler{
			requests: make(map[*yaml.Node]*TransferRequest),
		}
		for _, node := range tt.nodes {
			trh.HandleNode(node)
		}

		order := make([]*yaml.Node, 0)
		visited := make(map[*yaml.Node]bool, 0)
		for _, req := range trh.requests {
			trh.topologicalDfs(req, visited, &order)
		}

		// Compare order
		expected := []*yaml.Node{
			o3,
			o1,
			o2,
		}

		assert.Len(t, order, len(expected), "Should have the same number of nodes")
		assert.Equal(t, expected, order, "Should have the same order")
	}
}

func TestOptimizationFileComparisons(t *testing.T) {
	var tests = []struct {
		name         string
		inputFile    string
		expectedFile string
		handler      TRHandler
	}{
		// {
		// 	name:         "Test 1",
		// 	inputFile:    "testdata/optimization/basic.yaml",
		// 	expectedFile: "testdata/optimization/basic_expect.yaml",
		//  handler: TRHandler{
		// 		requests:   make(map[*yaml.Node]*TransferRequest),
		//  },
		// },
		// {
		// 	name:         "Test 2",
		// 	inputFile:    "testdata/optimization/nested.yaml",
		// 	expectedFile: "testdata/optimization/nested_expect.yaml",
		// 	handler: TRHandler{
		// 		requests:   make(map[*yaml.Node]*TransferRequest),
		// 		onlyUpdate: true,
		// 	},
		// },
		{
			name:         "Test 3",
			inputFile:    "testdata/optimization/complex_protected.yaml",
			expectedFile: "testdata/optimization/complex_protected_expect.yaml",
			handler: TRHandler{
				requests:      make(map[*yaml.Node]*TransferRequest),
				onlyUpdate:    true,
				protectOutput: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.ReadFile(tt.inputFile)
			if err != nil {
				t.Fatalf("Failed to read input file: %v", err)
			}

			b, err := os.ReadFile(tt.expectedFile)
			expected := string(b)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			var inNode yaml.Node
			err = yaml.Unmarshal(input, &inNode)
			if err != nil {
				t.Fatalf("Failed to unmarshal input file: %v", err)
			}

			trh := tt.handler
			trh.HandleRecursively(&inNode)
			trh.TransferAll()

			b, err = yaml.Marshal(&inNode)
			if err != nil {
				t.Fatalf("Failed to encode output: %v", err)
			}
			actual := string(b)
			os.WriteFile("testdata/optimization/out.yaml", b, 0644)

			assert.Equal(t, expected, actual, "Output node should match expected output")
		})
	}
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: value,
	}
}

func aliasNode(n *yaml.Node, anchor string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.AliasNode,
		Value: anchor,
		Alias: n,
	}
}

func toArray(m *yaml.Node) []string {
	arr := make([]string, 0, len(m.Content))

	for i := 0; i < len(m.Content); i += 1 {
		arr = append(arr, m.Content[i].Value)
	}

	return arr
}

func toArrayMultiple(m []*yaml.Node) []string {
	arr := make([]string, 0, len(m))

	for i := 0; i < len(m); i += 1 {
		arr = append(arr, m[i].Value, m[i].Anchor)
	}

	return arr
}
