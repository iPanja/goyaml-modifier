package modifier

import "gopkg.in/yaml.v3"

type TRHandler struct {
	requests map[*yaml.Node]TransferRequest
}

type TransferRequest struct {
	ins []*yaml.Node
	out *yaml.Node

	filter func(node *yaml.Node) bool
}

func MakeStandardTransferRequest() TransferRequest {
	return TransferRequest{
		ins:    make([]*yaml.Node, 0),
		out:    nil,
		filter: func(node *yaml.Node) bool { return true },
	}
}

// Transfer:
//
// For every node in each `in[i].Content`, attempt to move the value over to `out`
// Ins, out should be mapping nodes
//
//	If the key exists in multiple sources, only transfer it over if they all agree on the value
func (t *TransferRequest) Transfer() {
	var transfers map[*yaml.Node]string

	for _, in := range t.ins { // Loop over sources
		for _, n := range in.Content { // Loop over contents of source

		}
	}
}
