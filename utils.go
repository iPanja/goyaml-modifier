package modifier

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

/*
* File for one-off util methods
 */

func transferComments(in *yaml.Node, out *yaml.Node) {
	out.LineComment = in.LineComment
	out.HeadComment = in.HeadComment
	out.FootComment = in.FootComment
}

func getLookupValue(node *yaml.Node) string {
	if node.Kind == yaml.AliasNode {
		return node.Anchor
	}

	return node.Value
}

func getStructFieldKey(field reflect.StructField) string {
  k := field.Name
  k = strings.ToLower(k) // TODO: is this ok?

  tag := field.Tag.Get("yaml");
  if tag != "" {
    if tags := strings.Split(tag, ","); len(tags) > 0 {
      k = tags[0]
    }
  }

  return k
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	if node.Kind == yaml.AliasNode {
		return resolveAlias(node.Alias)
	}

	return node
}

func IsMergeKey(node *yaml.Node) bool {
  return node.Value == "<<" || node.Tag == "!!merge"
}

func IsInlineStructField(sf reflect.StructField) bool {
    tags, ok := sf.Tag.Lookup("yaml") 
    return ok && strings.Contains(tags, ",inline")
}
func IsOmitEmptyStructField(sf reflect.StructField) bool {
    tags, ok := sf.Tag.Lookup("yaml") 
    return ok && strings.Contains(tags, ",omitempty")
}
func ShouldSkipSliceEntry(v reflect.Value) bool {
  switch v.Kind() {
  case reflect.Slice, reflect.Array, reflect.Interface:
    return v.IsNil()
  }

  return false
}

func TransferAllComments(in *yaml.Node, out *yaml.Node) {
	if in == nil || out == nil {
		return
	}

	if in.Kind != out.Kind {
		fmt.Println("input node kind does not match outut node kind")
		return
	}

	transferComments(in, out)

	// Recursive
	if in.Kind == yaml.AliasNode {
		transferComments(in.Alias, out.Alias)
	} else if in.Kind == yaml.DocumentNode {
    return // Maybe I should still transfer all comments?
  }

	inc := 1
	if in.Kind == yaml.MappingNode {
		inc = 2
	}

	// Build lookup
	//  field name -> index
	//  This is more efficient, and also needed when contents are not an exact match (since we could have new fields)
	lookup := make(map[string]int, len(in.Content))
	for i := 0; i < len(in.Content); i += inc {
		primary := in.Content[i]
		lookup[getLookupValue(primary)] = i
	}

	// Iterate over out and apply these comments
	for i := 0; i < len(out.Content); i += inc {
		pOut := out.Content[i]
		pInIndex, ok := lookup[getLookupValue(pOut)]
		if !ok {
			continue // Not a match, no big deal
		}

		pIn := in.Content[pInIndex]

		TransferAllComments(pIn, pOut)
		// If we are looping through a map, we want to transfer the comments on the values as well
		if inc == 2 {
			sOut := out.Content[i+1]
			sIn := in.Content[pInIndex+1]
			TransferAllComments(sIn, sOut)
		}
	}
}

func determineNodeKind(v reflect.Value) yaml.Kind {
  switch v.Kind() {
  case reflect.Interface, reflect.Ptr:
    return determineNodeKind(v.Elem())
  case reflect.Slice, reflect.Array:
    return yaml.SequenceNode
  case reflect.Map, reflect.Struct:
    return yaml.MappingNode
  default:
    return yaml.ScalarNode
  }
}
