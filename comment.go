package modifier

import "gopkg.in/yaml.v3"

type Comment struct {
	line string
	head string
	foot string
}

func (c *Comment) Apply(node *yaml.Node) {
	node.LineComment = c.line
	node.HeadComment = c.head
	node.FootComment = c.foot
}
