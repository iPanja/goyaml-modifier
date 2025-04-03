package modifier

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/dprotaso/go-yit"
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

func (h *YAMLHandler) applyIterators(key *yaml.Node, value *yaml.Node, field reflect.StructField) {
  ytags := strings.Split(field.Tag.Get("yaml"), ",")
  htags := strings.Split(field.Tag.Get("mimo"), ",")

  for _, f := range h.NodeIterators {
    f(key, value, ytags, htags)
  }
}

// Import will read encode node into the interface
// NOTE: Entrypoint for YAMLHandler
func (h *YAMLHandler) ImportAndDecode(node *yaml.Node, v any) error {
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
func (h *YAMLHandler) Update(v any) error {
	// h.Update(
	// transferAllComments(in *yaml.Node, out *yaml.Node) -- Not needed since we have our own decoder
	//

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

  // TODO: can probably remove this
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("Expected a struct, got %v", val.Kind())
	}

	h.update(val, h.in, true) // TODO: use an internal h.shouldAdd variable

	return nil
}

// TODO: Build a flattened path to the node, to be passed to the iterator
func (h *YAMLHandler) update(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  if out.Kind == yaml.DocumentNode {
    h.update(val, out.Content[0], shouldAdd)
    return
  }

  if out.Kind == yaml.AliasNode {
    h.update(val, out.Alias, shouldAdd)
  }

  switch val.Kind() {
  case reflect.Struct:
    h.uStruct(val, out, shouldAdd)
  case reflect.Map:
    h.uMap(val, out, shouldAdd)
  case reflect.Int:
    uInt(val, out)
  case reflect.String:
    uString(val, out)
  case reflect.Bool:
    uBool(val, out)
  case reflect.Interface, reflect.Ptr:
    h.update(val.Elem(), out, shouldAdd)
  case reflect.Slice, reflect.Array:
    h.uSequence(val, out, shouldAdd)
    
  default:
    println("UNSUPPORTED TYPE!! ", val.Kind().String())
  }
}


// Indivial update methods

// node.Content is modified to reflect the state of val
// Explicit: true => out.Content will only contain k, v pairs present in (the map) val
//  All other nodes will be removed
func (h *YAMLHandler) uMap(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  // Build Lookup Table on struct
  // Map Key -> Value (reflect)
  lookup := make(map[string]reflect.Value, 0)
  buildMappingLookup(val, lookup)
 
  h.updateMap(yit.FromNodes(out.Content...), val, out, lookup, shouldAdd, true)

  nc := h.createRemainingNodes(lookup)
  out.Content = append(out.Content, nc...)
}

// Explicit: false => Existing nodes that were not in the struct will be left untouched (they will persist)
func (h *YAMLHandler) uStruct(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  // Build Lookup Table on struct
  // YAML Tag ( or Struct Field Name) -> Struct Field's Value
  lookup := make(map[string]reflect.Value, 0) // TODO: determine len for efficiency's sake
  buildMappingLookup(val, lookup)

  // Update nodes using lookup table
  it := yit.FromNodes(out.Content...)
  h.updateMap(it, val, out, lookup, shouldAdd, false)

  // Add in the new fields
  if !shouldAdd {
    return
  }

  nc := h.createRemainingNodes(lookup)
  out.Content = append(out.Content, nc...)
}



func (h *YAMLHandler) createRemainingNodes(lookup map[string]reflect.Value) []*yaml.Node {
  nc := make([]*yaml.Node, 0)
  for yamlKey, v := range lookup {
    nk := &yaml.Node {
      Kind: yaml.ScalarNode,
      Value: yamlKey,
    }

    nv := &yaml.Node {
      Kind: determineNodeKind(v),
      Value: "",
    }

    h.update(v, nv, true)
    // we could delete, but there isn't much point
    nc = append(nc, nk, nv)
  }

  return nc
}


// Utilizing a lookup table (node name -> struct field's value),
// Update the fields found in the node iterator and then remove them from the lookup table.
// Delete key, value pairs if the value is nil
// 
// Handles merge keys and aliases
func (h *YAMLHandler) updateMap(it yit.Iterator, val reflect.Value, out *yaml.Node, lookup map[string]reflect.Value, shouldAdd bool, explicit bool) {
  c := []*yaml.Node{}
  mvs := []*yaml.Node{} // Merge values

  for keyNode, ok := it(); ok; keyNode, ok = it() {
    value, _ := it()
    
    if IsMergeKey(keyNode) {
      // Store for later, explicit keys take priority
      mvs = append(mvs, value)
      c = append(c, keyNode, value)
      keyNode.Tag = "" // Remove !!merge in the output
      continue
    }

    if fieldValue, ok := lookup[keyNode.Value]; ok {
      if ShouldSkipField(fieldValue) {
        println("skipping ", keyNode.Value)
        continue
      }

      h.update(fieldValue, value, shouldAdd)
      delete(lookup, keyNode.Value)
      c = append(c, keyNode, value)
    } else if !explicit {
      // If explicit is disabled, we still want to keep nodes that are not getting updated
      c = append(c, keyNode, value)
    }
  }

  // Handle (multiple) merge keys
  for _, mv := range mvs {
    mi := FromMerge(mv)
    h.updateMap(mi, val, mv, lookup, shouldAdd, explicit)
  }

  out.Content = c
}

// node.Content is modified to reflect the state of val
//
// Overwrite nodes that share an index
// Delete excess nodes (delete out.Content[i] if i > val.Len())
// Delete nodes who's value is nil
func (h *YAMLHandler) uSequence(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  c := []*yaml.Node{}

  for i := range(max(len(out.Content), val.Len())) {
    inC := i < len(out.Content)
    inV := i < val.Len()

    if inC && inV {
      // Overwriting out.Content[i]
      e := val.Index(i)
      if ShouldSkipField(e) {
        continue
      }

      h.update(e, out.Content[i], shouldAdd)
      c = append(c, out.Content[i])
    } else if inC {
      // We have exhausted val, skip the remaining nodes from out.Content
      break
    } else {
      // Adding new node from val
      e := val.Index(i)
      if ShouldSkipField(e) {
        continue
      }

      n := &yaml.Node {
        Kind: determineNodeKind(e),
        Value: "",
      }

      h.update(e, n, true)
      c = append(c, n)
    }
  }

  out.Content = c
}

func uInt(val reflect.Value, out *yaml.Node) {
  out.Value = fmt.Sprintf("%d", val.Int())
}

func uString(val reflect.Value, out *yaml.Node) {
  out.Value = val.String()
}

func uBool(val reflect.Value, out *yaml.Node) {
  out.Value = fmt.Sprintf("%t", val.Bool())
}

