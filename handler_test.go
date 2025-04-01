package modifier

import (
	"testing"

	"gopkg.in/yaml.v3"
  "github.com/stretchr/testify/assert"
  "reflect"
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
        assert.Equal(t, len(n.Content), 3)
        assert.Equal(t, n.Content[1].Value, "LALALAL")
      },
    },
    {
      name: "Adding new node",
      seq: []any{"a", "b", "c", "d"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, len(n.Content), 4)
        assert.Equal(t, n.Content[3].Value, "d")
      },
    },
    {
      name: "Removing end node",
      seq: []any{"a", "b"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, len(n.Content), 2)
        assert.Equal(t, n.Content[0].Value, "a")
        assert.Equal(t, n.Content[1].Value, "b")
      },
    },
    {
      name: "Removing middle node via nil pointer",
      seq: []any{"a", nil, "c"},
      isValid: func(t *testing.T, n *yaml.Node) {
        assert.Equal(t, len(n.Content), 2)
        assert.Equal(t, n.Content[0].Value, "a")
        assert.Equal(t, n.Content[1].Value, "c")
      },
    },
  }


  for _, tt := range tests {
    n := nodeFn()
    s := tt.seq

    h := YAMLHandler{
      in: n,
    }

    h.uSequence(reflect.ValueOf(s), n, true)


    tt.isValid(t, n)
  }
}

func scalarNode(v string) *yaml.Node {
  return &yaml.Node {
    Kind: yaml.ScalarNode,
    Value: v,
  }
}
