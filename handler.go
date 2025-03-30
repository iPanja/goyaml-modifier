package modifier

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

// YAMLHandler provides an easy and convenient way of updating a YAML file,
//
//	with the simplicity of interacting with it through a struct
type YAMLHandler struct {
	in *yaml.Node

	// For sequences, only one would be set
	// ytags: `yaml:"..."`
	// htags: `yamlhandler:"..."`
	NodeIterators []func(key *yaml.Node, value *yaml.Node, ytags []string, htags []string) error
}

func NewYAMLHandler(node *yaml.Node) *YAMLHandler {
	return &YAMLHandler{
		in:            node,
		NodeIterators: make([]func(key *yaml.Node, value *yaml.Node, ytags []string, htags []string) error, 0),
	}
}

// AddNodeIterator stores a function that will be called on every node the handler either updates or adds inside of the Update() method
func (h *YAMLHandler) AddNodeIterator(iter func(key *yaml.Node, value *yaml.Node, ytags []string, htags []string) error) *YAMLHandler {
	h.NodeIterators = append(h.NodeIterators, iter)
	return h
}

// AddCommentHelper takes a map used to lookup comments that should be applied to the given node
// The keys (strings) can be node.Value, node.Anchor, the first yaml tag, or the first yaml handler tag
func (h *YAMLHandler) AddCommentHelper(lookup map[string]Comment, useYtags bool, useHtags bool) *YAMLHandler {
	iter := func(key *yaml.Node, value *yaml.Node, ytags []string, htags []string) error {
		lookupKeys := []string{
			getLookupValue(key),
		}

		if useYtags && len(ytags) > 0 {
			lookupKeys = append(lookupKeys, ytags[0])
		}
		if useHtags && len(htags) > 0 {
			lookupKeys = append(lookupKeys, htags[0])
		}

		for _, k := range lookupKeys {
			if c, ok := lookup[k]; ok {
				// TODO: Do not overwrite existing comments, instead append
				c.Apply(key)
				c.Apply(value)
			}
		}

		return nil // Can not fail
	}

	return h.AddNodeIterator(iter)
}

// Import will read encode node into the interface
// TODO: Should we (deep) clone the node?
// Entrypoint for YAMLHandler
func (h *YAMLHandler) Import(node *yaml.Node, v interface{}) error {
	h.in = node

	if err := node.Decode(v); err != nil {
		return err
	}

	return nil
}

// Update will update the originally imported yaml.Node with the new modifications from the struct
//
//	It also preforms other magic! (preserves comments, performs minimally-invase updates)
//
// Exitpoint for YAMLHandler
func (h *YAMLHandler) Update(v interface{}) error {
	// h.Update(
	// transferAllComments(in *yaml.Node, out *yaml.Node) -- Not needed since we have our own decoder
	//

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("Expected a struct, got %v", val.Kind())
	}

	update(val, h.in)

	return nil
}

// TODO: Build a flattened path to the node, to be passed to the iterator
func update(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  switch val.Kind() {
  case reflect.Struct:
    uStruct(val, out, shouldAdd)
  case reflect.Int:
    uInt(val, out)
  }
}


// Indivial update methods
func uStruct(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  t := val.Type()
  v := reflect.ValueOf(val)

  // Build Lookup Table on struct
  lookup := make(map[string]interface{}, 0) // TODO: determine len for efficiency's sake

  for i := range t.NumField() {
    var field reflect.StructField
    field = t.Field(i)
    var value reflect.Value
    value = v.Field(i)

    lookup[getStructFieldKey(field)] = value
	}

  // Attempt to update those fields present in the yaml.MappingNode
  // TODO: Make this smarter! Resolve alias nodes & handle merge keys
  for i := 0; i < len(out.Content); i += 2 {
    key := out.Content[i]
    if value, ok := lookup[key.Value]; !ok {
      nv := out.Content[i+1]

      update(reflect.ValueOf(value), nv, shouldAdd)
      delete(lookup, key.Value)
    }
  }

  // Add in the new fields
  if !shouldAdd {
    return
  }

  nc := make([]*yaml.Node, len(lookup))
  for i := range t.NumField() {
    field := t.Field(i)

    fieldKey := getStructFieldKey(field)
    if value, ok := lookup[fieldKey]; ok {
      nk := &yaml.Node {
        Kind: yaml.ScalarNode,
        Value: fieldKey,
      }

      var nv *yaml.Node
      update(reflect.ValueOf(value), nv, true)

      nc = append(nc, nk, nv)
    }
  }

  out.Content = append(out.Content, nc...)
}

func uSequence(val reflect.Value, out *yaml.Node) {
  // TODO: Determine how to handle this case, or don't...
}

func uInt(val reflect.Value, out *yaml.Node) {
  out.Value = fmt.Sprintf("%d", val.Int())
}

func uString(val reflect.Value, out *yaml.Node) {
  out.Value = val.String()
}

func uBool(val reflect.Value, out *yaml.Node) {
  out.Value = fmt.Sprintf("%t", val.Bool)
}
