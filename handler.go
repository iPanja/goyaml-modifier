package modifier

import (
	"fmt"
	"reflect"

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
	NodeIterators    []func(node *yaml.Node) error
	overrideExplicit bool // Ensure maps are only updated, and no nodes are removed
}

func NewYAMLHandler(node *yaml.Node) *YAMLHandler {
	return &YAMLHandler{
		in:            node,
		NodeIterators: make([]func(node *yaml.Node) error, 0),
	}
}

// AddNodeIterator stores a function that will be called on every node the handler either updates or adds inside of the Update() method
func (h *YAMLHandler) AddNodeIterator(iter func(node *yaml.Node) error) *YAMLHandler {
	h.NodeIterators = append(h.NodeIterators, iter)
	return h
}

// Import will read encode node into the interface
// NOTE: Entrypoint for YAMLHandler
func ImportAndDecode(node *yaml.Node, v any) (*YAMLHandler, error) {
	h := YAMLHandler{}
	h.in = node

	if err := node.Decode(v); err != nil {
		return nil, err
	}

	return &h, nil
}

func (h *YAMLHandler) applyIterators(node *yaml.Node) {
	for _, iter := range h.NodeIterators {
		if err := iter(node); err != nil {
			panic(err)
		}
	}
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

	h.update(val, h.in)

	return nil
}

// TODO: Build a flattened path to the node, to be passed to the iterator
func (h *YAMLHandler) update(val reflect.Value, out *yaml.Node) {
	if out.Kind == yaml.DocumentNode {
		h.update(val, out.Content[0])
		return
	}

	if out.Kind == yaml.AliasNode {
		h.update(val, out.Alias)
	}

	h.applyIterators(out)

	switch val.Kind() {
	case reflect.Struct:
		h.uStruct(val, out)
	case reflect.Map:
		h.uMap(val, out)
	case reflect.Int:
		uInt(val, out)
	case reflect.String:
		uString(val, out)
	case reflect.Bool:
		uBool(val, out)
	case reflect.Interface, reflect.Ptr:
		h.update(val.Elem(), out)
	case reflect.Slice, reflect.Array:
		h.uSequence(val, out)

	default:
		println("UNSUPPORTED TYPE!! ", val.Kind().String())
	}
}

// Indivial update methods

// node.Content is modified to reflect the state of val
// Explicit: true => out.Content will only contain k, v pairs present in (the map) val
//
//	All other nodes will be removed
func (h *YAMLHandler) uMap(val reflect.Value, out *yaml.Node) {
	// Build Lookup Table on struct
	// Map Key -> Value (reflect)
	lookup := make(map[string]reflect.Value, 0)
	buildMappingLookup(val, lookup)

	h.updateMap(yit.FromNodes(out.Content...), val, out, lookup, true)

	nc := h.createRemainingNodes(lookup)
	out.Content = append(out.Content, nc...)
}

// Explicit: false => Existing nodes that were not in the struct will be left untouched (they will persist)
func (h *YAMLHandler) uStruct(val reflect.Value, out *yaml.Node) {
	// Build Lookup Table on struct
	// YAML Tag ( or Struct Field Name) -> Struct Field's Value
	lookup := make(map[string]reflect.Value, 0) // TODO: determine len for efficiency's sake
	buildMappingLookup(val, lookup)

	// Update nodes using lookup table
	it := yit.FromNodes(out.Content...)
	h.updateMap(it, val, out, lookup, false)

	// Add in the new fields
	nc := h.createRemainingNodes(lookup)
	out.Content = append(out.Content, nc...)
}

func (h *YAMLHandler) createRemainingNodes(lookup map[string]reflect.Value) []*yaml.Node {
	nc := make([]*yaml.Node, 0)
	for yamlKey, v := range lookup {
		nk := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: yamlKey,
		}

		nv := &yaml.Node{
			Kind:  determineNodeKind(v),
			Value: "",
		}

		h.update(v, nv)
		// we could delete, but there isn't much point
		nc = append(nc, nk, nv)
	}

	return nc
}

// Utilizing a lookup table (node name -> struct field's value),
// Update the fields found in the node iterator and then remove them from the lookup table.
// Delete key, value pairs if the value is nil
// Explicit (if true): only keep nodes that are present in lookup, all others will be removed
//
// Does NOT handle merge keys and aliases
func (h *YAMLHandler) updateMap(it yit.Iterator, val reflect.Value, out *yaml.Node, lookup map[string]reflect.Value, explicit bool) {
	c := []*yaml.Node{}

	if h.overrideExplicit {
		explicit = false
	}

	for keyNode, ok := it(); ok; keyNode, ok = it() {
		value, _ := it()

		if fieldValue, ok := lookup[keyNode.Value]; ok {
			if ShouldSkipField(fieldValue) {
				continue
			}

			h.update(fieldValue, value)
			delete(lookup, keyNode.Value)
			c = append(c, keyNode, value)
		} else if !explicit {
			// If explicit is disabled, we still want to keep nodes that are not getting updated
			c = append(c, keyNode, value)
		}
	}

	out.Content = c
}

// node.Content is modified to reflect the state of val
//
// Overwrite nodes that share an index
// Delete excess nodes (delete out.Content[i] if i > val.Len())
// Delete nodes who's value is nil
func (h *YAMLHandler) uSequence(val reflect.Value, out *yaml.Node) {
	c := []*yaml.Node{}

	for i := range max(len(out.Content), val.Len()) {
		inC := i < len(out.Content)
		inV := i < val.Len()

		if inC && inV {
			// Overwriting out.Content[i]
			e := val.Index(i)
			if ShouldSkipField(e) {
				continue
			}

			h.update(e, out.Content[i])
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

			n := &yaml.Node{
				Kind:  determineNodeKind(e),
				Value: "",
			}

			h.update(e, n)
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
