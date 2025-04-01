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

  if !val.IsValid() {
    println("NOT VALID!!! (idk)")
    return
  }

  switch val.Kind() {
  case reflect.Struct:
    h.uStruct(val, out, shouldAdd)
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
func (h *YAMLHandler) uStruct(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  t := val.Type()

  // Build Lookup Table on struct
  // YAML Tag ( or Struct Field Name) -> Struct Field's Value
  lookup := make(map[string]reflect.Value, 0) // TODO: determine len for efficiency's sake

  fields := reflect.VisibleFields(t)
  for _, field := range fields{
    v := val.FieldByName(field.Name)
    if tags, ok := field.Tag.Lookup("yaml"); ok && strings.Contains(tags, ",inline") && v.Kind() == reflect.Map {
      // inline (map?)
      // TODO: can an inline field be anything other than a map
      iter := v.MapRange()
      for iter.Next() {
        mk := iter.Key()
        mv := iter.Value()

        if IsOmitEmptyStructField(field) && mv.IsZero() {
          continue
        }

        lookup[mk.String()] = mv
      }
    } else {
      // Normal struct field
      if IsOmitEmptyStructField(field) && v.IsZero() {
        continue
      }
      lookup[getStructFieldKey(field)] = v
    } 
	}

  // Update nodes that are found in the struct
  it := yit.FromNodes(out.Content...)
  h.updateMap(it, val, lookup, shouldAdd)

  // Add in the new fields
  if !shouldAdd {
    return
  }

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
    // we could delete, but there isn't much point unless we return the lookup map or something
    nc = append(nc, nk, nv)
  }

  out.Content = append(out.Content, nc...)
}


// Utilizing a lookup table (node name -> struct field's value),
// Update the fields found in the node iterator and then remove them from the lookup table.
// 
// Handles merge keys and aliases
func (h *YAMLHandler) updateMap(it yit.Iterator, val reflect.Value, lookup map[string]reflect.Value, shouldAdd bool) {
  var mi yit.Iterator

  for keyNode, ok := it(); ok; keyNode, ok = it() {
    value, _ := it()
    
    if IsMergeKey(keyNode) {
      // Store for later, explicit keys take priority
      mi = FromMerge(value)
    }

    if fieldValue, ok := lookup[keyNode.Value]; ok {
      h.update(fieldValue, resolveAlias(value), shouldAdd)
      delete(lookup, keyNode.Value)
    }
  }

  // NOTE: Perhaps call trimContents(...) here?
  // ISSUE: We don't have access to the underlying node since we only have the iterator...

  if mi != nil {
    h.updateMap(mi, val, lookup, shouldAdd)
    // NOTE: Perhaps call trimContents(...) here
    // HACK: Create a crawler at the end of the update() method 
    //  that can handle removing nil cases appropriately
    //  depending on the reflect.Type (map, slice, scalar, etc)
  }
}

// NOTE: To properly remove an entry, set it to nil so we can still utilize the order/len
func (h *YAMLHandler) uSequence(val reflect.Value, out *yaml.Node, shouldAdd bool) {
  // Update in terms of order
  length := min(len(out.Content), val.Len())
  out.Content = out.Content[:length] // Potentially remove excess YAML nodes
  
  for i := range length {
    e := val.Index(i)

    if ShouldSkipSliceEntry(e) { // Allow explicit deletion of entries via nil pointer
      continue // Why waste time...
    }

    h.update(e, out.Content[i], shouldAdd)
  }


  // Add new content
  if !shouldAdd {
    return
  }

  for i := len(out.Content); i < val.Len(); i++ {
    e := val.Index(i)

    n := &yaml.Node {
      Kind: determineNodeKind(e),
      Value: "",
    }

    h.update(e, n, true)
    out.Content = append(out.Content, n)
  }

  // Deal with node removal (if one is set to nil)
  trimContents(val, out)
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

// trimContents will remove entries from node.Content if they are a nil pointer
// NOTE: To be depreciated with the implementation of trimNilNodes()
func trimContents(val reflect.Value, out *yaml.Node) {
  c := []*yaml.Node{}
  for i, n := range out.Content {
    e := val.Index(i)

    if !ShouldSkipSliceEntry(e) {
      c = append(c, n)
    }
  }

  // Only modify if we need to
  if len(c) != len(out.Content) {
    out.Content = c
  }
}

// trimNilNodes will recursively iterate through the node and remove those who's value are nil
//
//  - scalar: nil (scalar)
//  - key: nil (map)
//  - nil (slice)
func trimNilNodes(node *yaml.Node) {
  // TODO: IMPLEMENT & CALL
}
