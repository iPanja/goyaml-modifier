package modifier

import (
	"testing"

	"gopkg.in/yaml.v3"
  "github.com/stretchr/testify/assert"
  "reflect"
  "github.com/dprotaso/go-yit"

)


func TestHandlerAlias(t *testing.T) {
	type emb struct {
		Name string
		B    string
	}
	type config struct {
    Thingy emb
	}

  data := `doc:
  dummy: &var original
  thingy:
    name: *var
    b: idk
`

  result := `dummy: &var please update alias value!
thingy:
    name: *var
    b: idk
`
  
  // Access struct we are using
  c := config{}
  handler := loadAndSetup(t, data, &c)


  // Marshal & Test Result
  c.Thingy.Name = "please update alias value!"
  handler.Update(&c)

  b, err := yaml.Marshal(handler.in)
  if err != nil {
    t.Error(err)
  }

  assert.Equal(t, result, string(b))
}

func TestStandardHandler(t *testing.T)  {
  type emb struct {
    Name string
    B string
  }

  type config struct {
    Version string
    Year int `yaml:"release_year"`
    IsLive bool `yaml:"is_live"`
    Seq []emb
    SpecificEmb emb `yaml:"specific_emb"`
  }

  stdData := `doc:
  version: old
  release_year: 2020
  is_live: true
  seq:
    - Name: First
      B: something
    - Name: Second
      B: another
  specific_emb:
    name: Third
    b: final
  `

  result := `version: newer
release_year: 2025
is_live: false
seq:
    - Name: First
      B: something
    - Name: Second
      B: another
specific_emb:
    name: NEW NAME
    b: final
`

  // Access struct we are using
  c := config{}
  handler := loadAndSetup(t, stdData, &c)

  // Modify struct
  c.Version = "newer"
  c.Year = 2025
  c.IsLive = false
  c.Seq[1].B = "changed"
  c.SpecificEmb.Name = "NEW NAME"

  // Marshal & Test Result
  handler.Update(&c)

  b, err := yaml.Marshal(handler.in)
  if err != nil {
    t.Error(err)
  }

  assert.Equal(t, result, string(b))
}

func TestInlineHandler(t *testing.T)  {
  type emb struct {
    Name string
    B string
  }

  type config struct {
    Version string
    Year int `yaml:"release_year"`
    Other map[string]any `yaml:",inline"`
  }

  stdData := `doc:
  version: old
  release_year: 2020
  is_live: true
  seq:
    - Name: First
      B: something
    - Name: Second
      B: another
  a: b
  c: d
  `

  result := `version: newer
release_year: 2025
is_live: false
seq:
    - Name: First
      B: something
    - Name: Second
      B: another
a: b
c: eee
`

  c := config{}
  handler := loadAndSetup(t, stdData, &c)

  // Modify struct
  c.Version = "newer"
  c.Year = 2025
  c.Other["is_live"] = false
  c.Other["c"] = "eee"

  // Marshal & Test Result
  handler.Update(&c)

  b, err := yaml.Marshal(handler.in)
  if err != nil {
    t.Error(err)
  }

  assert.Equal(t, result, string(b))
}


// TODO: Test and create logic for deleting nodes (via nil & when the slice is shorter than len(node.Content))
func TestSeqHandler(t *testing.T)  {
  type config struct {
    Version string
    Names []string
  }

  stdData := `doc:
  version: old
  names:
    - A
    - B
  `

  result := `version: newer
names:
    - AAA
    - B
    - C
`


  c := config{}
  handler := loadAndSetup(t, stdData, &c)

  // Modify struct
  c.Version = "newer"
  c.Names = append(c.Names, "C")
  c.Names[0] = "AAA"

  // Marshal & Test Result
  handler.Update(&c)

  b, err := yaml.Marshal(handler.in)
  if err != nil {
    t.Error(err)
  }

  print(string(b))
  assert.Equal(t, result, string(b))
}

func loadAndSetup(t *testing.T, yamlData string, v any) *YAMLHandler {
  var node yaml.Node

  if err := yaml.Unmarshal([]byte(yamlData), &node); err != nil {
    t.Error(err)
  }

  // Access struct we are using
  contents := node.Content[0].Content[1]
  handler := YAMLHandler{}
  handler.ImportAndDecode(contents, v)

  return &handler

}

// TODO: Implement & Test on Mapping, Structs
// Also test nil/removing entries on both of those types
func TestUSequence(t *testing.T) {
  nodeFn := func() *yaml.Node {
    return &yaml.Node {
      Kind: yaml.SequenceNode,
      Content: []*yaml.Node {
        scalarNode("a"),
        scalarNode("b"),
        scalarNode("c"),
      },
    }
  }

  var tests = []struct {
    name string
    seq []any
    isValid func(t *testing.T, n *yaml.Node)
  }{
    {
      name: "Standard upate",
      seq: []any{"a", "LALALAL", "c"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, 3, len(n.Content))
        assert.Equal(t, "LALALAL", n.Content[1].Value)
      },
    },
    {
      name: "Adding new nodes",
      seq: []any{"a", "b", "c", "d", "e"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, 5, len(n.Content), 5)
        assert.Equal(t, "d", n.Content[3].Value)
        assert.Equal(t, "e", n.Content[4].Value)
      },
    },
    {
      name: "Removing end node",
      seq: []any{"a", "b"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, 2, len(n.Content))
        assert.Equal(t, "a", n.Content[0].Value)
        assert.Equal(t, "b", n.Content[1].Value)
      },
    },
    {
      name: "Removing middle node via nil pointer",
      seq: []any{"a", nil, "c"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, 2, len(n.Content))
        assert.Equal(t, "a", n.Content[0].Value)
        assert.Equal(t, "c", n.Content[1].Value)
      },
    },
    {
      name: "Remove all nodes without nil",
      seq: []any{},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, 0, len(n.Content))
      },
    },
    {
      name: "Remove multiple nodes with nil and add new",
      seq: []any {nil, nil, nil, "a"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, 1, len(n.Content))
        assert.Equal(t, "a", n.Content[0].Value)
      },
    },
  }


  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      n := nodeFn()
      s := tt.seq

      h := YAMLHandler{
        in: n,
      }

      h.uSequence(reflect.ValueOf(s), n, true)


      tt.isValid(t, n)
    })
  }
}

func TestUMap(t *testing.T) {
  nodeFn := func() *yaml.Node {
    return &yaml.Node {
      Kind: yaml.MappingNode,
      Content: []*yaml.Node {
        scalarNode("a"),
        scalarNode("b"),
        scalarNode("c"),
        scalarNode("d"),
      },
    }
  }

  var tests = []struct {
    name string
    mapping map[string]any
    isValid func(t *testing.T, n *yaml.Node)
  }{
    {
      name: "Standard upate",
      mapping: map[string]any {
        "a": "new",
        "e": "f",
      },
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, []any{"a", "new", "e", "f"}, mapValues(yit.FromNodes(n.Content...)))
      },
    },
    {
      name: "Remove via nil",
      mapping: map[string]any {
        "a": "new",
        "c": nil,
      },
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, []any{"a", "new"}, mapValues(yit.FromNodes(n.Content...)))
      },
    },
    {
      name: "Adding a bunch",
      mapping: map[string]any {
        "new": "thing",
        "a": "b",
        "c": nil,
        "another": "pair",
        "wahaha": "hi",
      },
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.ElementsMatch(t, []any{"new", "thing", "a", "b", "another", "pair", "wahaha", "hi"}, mapValues(yit.FromNodes(n.Content...)))
      },
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      n := nodeFn()
      m := tt.mapping

      h := YAMLHandler{
        in: n,
      }

      h.uMap(reflect.ValueOf(m), n, true)

      tt.isValid(t, n)
    })
  }
}

func scalarNode(v string) *yaml.Node {
  return &yaml.Node {
    Kind: yaml.ScalarNode,
    Value: v,
  }
}

func mapValues(it yit.Iterator) []any {
  result := []any{}

  for key, ok := it(); ok; key, ok = it() {
    value, _ := it()

    if IsMergeKey(key) {
      result = append(result, mapValues(FromMerge(value)))
    } else {
      result = append(result, key.Value, value.Value)
    }
  }

  return result
}
