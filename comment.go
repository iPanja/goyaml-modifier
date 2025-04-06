package modifier

type Comment struct {
	line string
	head string
	foot string
}

// func (c *Comment) Apply(node *yaml.Node) {
// 	node.LineComment = c.line
// 	node.HeadComment = c.head
// 	node.FootComment = c.foot
// }

// // AddCommentHelper takes a map used to lookup comments that should be applied to the given node
// // The keys (strings) can be node.Value, node.Anchor, the first yaml tag, or the first yaml handler tag
// func (h *YAMLHandler) AddCommentHelper(lookup map[string]Comment, useYtags bool, useHtags bool) *YAMLHandler {
// 	iter := func(key *yaml.Node, value *yaml.Node, ytags []string, htags []string) error {
// 		lookupKeys := []string{
// 			getLookupValue(key),
// 		}

// 		if useYtags && len(ytags) > 0 {
// 			lookupKeys = append(lookupKeys, ytags[0])
// 		}
// 		if useHtags && len(htags) > 0 {
// 			lookupKeys = append(lookupKeys, htags[0])
// 		}

// 		for _, k := range lookupKeys {
// 			if c, ok := lookup[k]; ok {
// 				// TODO: Do not overwrite existing comments, instead append
// 				c.Apply(key)
// 				c.Apply(value)
// 			}
// 		}

// 		return nil // Can not fail
// 	}

// 	return h.AddNodeIterator(iter)
// }
