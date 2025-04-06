package modifier

import (
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
	}

	var tests = []struct {
		name    string
		nodes   []*yaml.Node
		isValid func(t *testing.T, trh *TRHandler)
	}{
		{
			name: "Valid Mapping Node",
			nodes: []*yaml.Node{
				{
					Kind:    yaml.MappingNode,
					Content: []*yaml.Node{},
					Anchor:  "anchor1",
				},
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{
							Kind: yaml.MappingNode,
							Content: []*yaml.Node{
								scalarNode("<<"),
								aliasNode(o, "anchor1"),
							},
						},
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
				assert.Len(t, trh.requests, 1, "Should have 1 requests")
				tr := trh.requests[o]
				assert.Len(t, tr.ins, 2, "Should have 2 sources")
				assert.Equal(t, o, tr.out, "Should have the same output node")
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
