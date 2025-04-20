package modifier

import (
	"slices"

	"gopkg.in/yaml.v3"
)

type TRHandler struct {
	// Perhaps create a dependency graph to run through these in the correct order
	requests map[*yaml.Node]*TransferRequest

	// if true, new nodes will not be created within the out node
	onlyUpdate bool

	// If true, the output node will not be modified.
	// This is useful if output was directly modified by the user, and so we want to protect the values inside of it from being overwritten
	protectOutput bool

	// More granular than protectOutput
	protectedNodes []*yaml.Node

	// This field will override the default of each transfer request
	filter func(k *yaml.Node, v *yaml.Node) bool
}

func (h *TRHandler) HandleRecursively(node *yaml.Node) {
	if node.Kind == yaml.MappingNode {
		h.HandleNode(node)
	}

	for _, child := range node.Content {
		h.HandleRecursively(child)
	}
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
	// Transfer over safety checks
	for _, k := range order {
		if req, ok := h.requests[k]; ok {
			req.onlyUpdate = h.onlyUpdate
			req.protectOutput = h.protectOutput
			req.protectedNodes = h.protectedNodes
			req.filter = h.filter
			req.Transfer()
		}
	}

	return nil
}

type TransferRequest struct {
	ins []*yaml.Node
	out *yaml.Node

	// if true, new nodes will not be created within the out node
	onlyUpdate bool

	// If true, the output node will not be modified.
	// This is useful if output was directly modified by the user, and so we want to protect the values inside of it from being overwritten
	protectOutput bool

	// More granular than protectOutput
	protectedNodes []*yaml.Node

	filter func(k *yaml.Node, v *yaml.Node) bool
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

	nested := make(map[string]TransferRequest, 0) // key.Value -> handler

	// Populate transfers
	for _, in := range t.ins {
		for i := 0; i < len(in.Content); i += 2 {
			k := in.Content[i]
			v := in.Content[i+1] // TODO: Resolve!

			if IsMergeKey(k) || v.Alias != nil {
				// TODO: do we want to follow these?
				continue
			}

			// Handle nested transfer requests
			if v.Kind == yaml.MappingNode || v.Kind == yaml.SequenceNode {
				if tr, ok := nested[k.Value]; ok {
					// We have already created a transfer request for this node
					tr.ins = append(tr.ins, v)
					nested[k.Value] = tr
				} else {
					// Create a new transfer request for this node
					if out, ok := outLookup[k.Value]; ok {
						tr := MakeStandardTransferRequest(out)
						tr.ins = append(tr.ins, v)
						nested[k.Value] = tr
					} else {
						// We need to create a new node within the out node
						// Only do this if t.onlyUpdate == false
						// TODO:
					}
				}
				continue
			}

			// Normal scalars
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
			if t.protectOutput || slices.Contains(t.protectedNodes, outNode) {
				// For protected nodes, we are not overwriting the value
				// And we only then cleanup the input nodes if they agere with the output
				if outNode.Value != p.val.Value {
					continue
				}
			}

			// TODO: move comments over as well
			// Deepcopy?
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

	// Handle nested transfer requests
	for s, tr := range nested {
		// Transfer
		tr.onlyUpdate = t.onlyUpdate
		tr.protectOutput = t.protectOutput
		tr.protectedNodes = t.protectedNodes
		tr.filter = t.filter
		tr.Transfer()

		// Nested blocks do not have a merge key, so there is a chance we delete all of the content
		for _, in := range t.ins {
			if n := getMapValue(in, s); n != nil && len(n.Content) == 0 {
				removeKeyValuePair(in, s)
			}
		}
	}
}
