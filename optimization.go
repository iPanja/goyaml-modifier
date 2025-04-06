package modifier

import (
	"gopkg.in/yaml.v3"
)

type TRHandler struct {
	// Perhaps create a dependency graph to run through these in the correct order
	requests map[*yaml.Node]*TransferRequest

	// This field will override the default of each transfer request
	onlyUpdate bool
	// This field will override the default of each transfer request
	filter func(k *yaml.Node, v *yaml.Node) bool
}

// m: MappingNode
func (h *TRHandler) HandleNode(m *yaml.Node) {
	if m.Kind != yaml.MappingNode {
		return
	}

	// Create a new transfer request since it has an anchor
	// This node is the `out`
	if m.Anchor != "" {
		if _, ok := h.requests[m]; !ok {
			r := MakeStandardTransferRequest(m)
			h.requests[m] = &r
		}
	}

	// Add as a source node if it has a merge key
	if mergeValue := getMapValue(m, "<<"); mergeValue != nil {
		it := FromMergeJustAliases(mergeValue)
		for a, ok := it(); ok; a, ok = it() {
			if req, ok := h.requests[a.Alias]; ok {
				// The underlying alias already has a request handler
				req.ins = append(req.ins, m)
				h.requests[a.Alias] = req
			} else {
				// The underlying alisas needs to have a handler created
				// a.Alias is the output node
				// m is the input node
				r := MakeStandardTransferRequest(a.Alias)
				r.ins = append(r.ins, m)
				h.requests[a.Alias] = &r
			}
		}
	}
}

// topologicalSort will sort the nodes in a topological order
//
// There are no cycles in YAML, so we don't need to worry about that
func (h *TRHandler) topologicalDfs(request *TransferRequest, visited map[*yaml.Node]bool, order *[]*yaml.Node) {
	if visited[request.out] {
		return
	}
	visited[request.out] = true

	// Process dependencies (request.ins)
	for _, in := range request.ins {
		if req, ok := h.requests[in]; ok {
			h.topologicalDfs(req, visited, order)
		}
	}

	*order = append(*order, request.out)
}

func (h *TRHandler) TransferAll() error {
	order := make([]*yaml.Node, 0)
	visited := make(map[*yaml.Node]bool, 0)
	for _, req := range h.requests {
		h.topologicalDfs(req, visited, &order)
	}

	// Reverse the order
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	// Transfer in the correct order
	for _, k := range order {
		if req, ok := h.requests[k]; ok {
			req.onlyUpdate = h.onlyUpdate
			req.filter = h.filter
			req.Transfer()
		}
	}

	return nil
}

type TransferRequest struct {
	ins []*yaml.Node
	out *yaml.Node

	onlyUpdate bool
	filter     func(k *yaml.Node, v *yaml.Node) bool
}

func MakeStandardTransferRequest(out *yaml.Node) TransferRequest {
	return TransferRequest{
		ins:        make([]*yaml.Node, 0),
		out:        out,
		onlyUpdate: false,
		filter:     func(k *yaml.Node, v *yaml.Node) bool { return true },
	}
}

// Transfer:
//
// For every node in each `in[i].Content`, attempt to move the value over to `out`
// Ins, out should be mapping nodes
//
//	If the key exists in multiple sources, only transfer it over if they all agree on the value
type pair struct {
	key   *yaml.Node
	val   *yaml.Node
	agree bool
}

func (t *TransferRequest) Transfer() {
	transfers := make(map[string]*pair, 0) // key.Value -> (Key, Value) node
	outLookup := buildLookup(t.out)        // key.Value -> Value node

	// Populate transfers
	for _, in := range t.ins {
		for i := 0; i < len(in.Content); i += 2 {
			k := in.Content[i]
			v := in.Content[i+1]

			if IsMergeKey(k) || v.Alias != nil {
				// TODO: do we want to follow these?
				continue
			}

			if p, ok := transfers[k.Value]; ok {
				// Only transfer if all sources agree on the value
				if p.val.Value != v.Value {
					transfers[k.Value].agree = false
					continue
				}
			} else {
				// First appearance
				transfers[k.Value] = &pair{k, v, true}
			}
		}
	}

	// Move successful transfers
	for k, p := range transfers {
		if !p.agree || (t.filter != nil && !t.filter(p.key, p.val)) {
			continue
		}

		// TODO: Filter here?
		if outNode, ok := outLookup[k]; ok {
			// We are updating an existing node
			outNode.Value = p.val.Value
		} else if !t.onlyUpdate {
			// Add a new node, IF we are not only updating
			// Deepcopy?
			t.out.Content = append(t.out.Content, p.key, p.val)
		} else {
			continue
		}

		// Remove the key from the source nodes
		for _, in := range t.ins {
			removeKeyValuePair(in, k)
		}
	}
}
