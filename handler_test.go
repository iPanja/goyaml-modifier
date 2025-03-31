package modifier

import (
	"testing"

  "fmt"
	"gopkg.in/yaml.v3"
)

const yd = `doc:
  version: newest
  year: 2025
  random_bool: true
  seq:
    - name: First
      b: ok
      dummy: entry
    - name: Second
      b: something else
`

func TestHandler(t *testing.T) {
	type emb struct {
		Name string
		B    string

		//Unused map[string]interface{} `yaml:",inline"`
	}
	type config struct {
    Version    string `yaml:"version"`
		Year       int
		RandomBool bool `yaml:"random_bool"`
		Seq        []emb
    InlineStuff map[string]any `yaml:",inline"`
    NewEntry string `yaml:"new_entry,omitempty"`
	}

  
  // EXTRACT TEST YAML
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yd), &node); err != nil {
		println(err)
		return
	}
  println("Unmarshalled")

  var Doc struct {
    C config `yaml:"doc"`
    // Unused map[string]interface{} `yaml:",inline"`
    // TODO: Handle inline ^^
  }

  contents := node.Content[0].Content[1] // Take care of document node until the library can handle it...
	handler := YAMLHandler{}
	handler.ImportAndDecode(contents, &Doc.C)


	println("Found version: ", Doc.C.Version)

  // MODIFY STRUCT & RE-UPDATE
  Doc.C.Version = "EVEN NEWER"
  Doc.C.Year = 2029
  Doc.C.InlineStuff = map[string]any {
    "a": "b",
    "c": "d",
  }
  Doc.C.NewEntry = "testing!!"
  handler.Update(&Doc.C)

  fmt.Println("\nRESULTS:")
  if b, err := yaml.Marshal(handler.in); err != nil {
    println(err)
    return
  } else {
    println(string(b))
  }
}
