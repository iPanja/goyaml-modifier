package modifier

import "gopkg.in/yaml.v3"

type TRHandler struct {
	// Perhaps create a dependency graph to run through these in the correct order
	requests map[*yaml.Node]*TransferRequest
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
				r := MakeStandardTransferRequest(m)
				h.requests[a.Alias] = &r
				h.requests[a.Alias].ins = append(h.requests[a.Alias].ins, m)
			}
		}
	}
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
