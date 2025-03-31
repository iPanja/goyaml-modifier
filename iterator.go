package modifier

import (
  yit "github.com/dprotaso/go-yit"
	"gopkg.in/yaml.v3"
)


// type NodeIterator struct {
//   yit.Iterator
// }


// FromMerge allows you to iterate over the contents of a merge key's nodes
func FromMerge(v *yaml.Node) yit.Iterator {
  // <<: *alias
  if v.Kind == yaml.AliasNode {
    return yit.FromNodes(v.Content...)
  }

  // <<: [*alias_one, *alias_two, ...]
  its := make([]yit.Iterator, len(v.Content))
  for _, a := range v.Content {
    its = append(its, yit.FromNodes(a.Content...))
  }

  return yit.FromIterators(its...)
}
