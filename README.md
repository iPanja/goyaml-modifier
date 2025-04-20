# Go YAML Modifier (MIMO)

A minimally invasive YAML modifier.

## Purpose

This tool aims to solve the issue of modifying YAML files while preserving their original structure (as much as possible).

It provides a custom "encoder", allowing the usage of go data types without losing the normal YAML structure (aliases, merges, etc) you would when encoding back.

## Usage

1. Load in your `yaml.Node` into a data type of your choosing:

```go
var v = struct {
  Name string `yaml:"my_name"`
  Year int

  Unused map[string]any `yaml:",inline"`
}


h := YAMLHandler{}
h.ImportAndDecode(myDataNode, &v)
```

2. Modify your internal data

```go
v.Name = "New name"
v.Year += 1
v.Unused["random_new_entry"] = "hello!"
```

3. Save it back

```go
h.Update(v)
h.Optimize()
```

### Here are possible outputs that could be generated based on the input

#### Example 1

```yaml
var: &name Somebody
data:
  name: *name
  year: 2020
```

```yaml
var: &name New name
data:
  name: *name
  year: 2021
  random_new_entry: hello!
```

#### Example 2

```yaml
other: &ref
  name: Somebody
another: &ref2
  name: defined later, does not have precedence
  Year: 2020
data:
  <<: [*ref, *ref2]
```

```yaml
other: &ref
  name: New name
another: &ref2
  name: defined later, does not have precedence
  Year: 2021
data:
  <<: [*ref, *ref2]
  new_random_entry: hello!

```

## Documentation

### YAMLHandler
Used to update a `yaml.Node` as an alternative to utilizing `node.Encode()`.  It will update existing nodes in order to preserve the rest of the node that would usually be lost by directly encoding.

It also provides an accessible way to call custom functions on nodes that do get updated:
`func(node *yaml.Node, path []string) error`.

Additionally, it tracks the nodes that get modified and exposes an easy `Optimize()` method that refactors the contents of a map (and nested structures) via `TRHandler` and `TransferRequest`.

### TRHandler
Automatically orchestrates multiple TransferRequests under the hood, and abstracts the process of creating them.

Usage:
 - `HandleRecursively` - Scan a node and all of its children recursively, calling `HandleNode()` on them.
 - `HandleNode` - Given a map, if it has an anchor then create a transfer request with it as the output node. If the map contains one or more merge keys, create the appropriate requests.

### TransferRequest
- Attempt to move values from the `input` nodes (mapping or sequence node) to the `output` node only if all the inputs agree on a value
- HandlerOptions:
  - `onlyUpdate` - Do not add new nodes to `output`, only refactor nodes that exist within the output node.
  - `protectOutput` - Do not modify the output node or its contents whatsoever. So, only remove duplicate, redundent nodes from inputs
  - `protectNodes` - Do not modify these specific nodes. This is more granular than `protectOutput`(which protects all of `output.Contents`).

The last two options are meant to be used when you directly modify the `output` block. In that scenario, you may not want to accidentally overwrite the value with those from `inputs`. The `YAMLHandler` automatically keeps track of the specific nodes you modify and sets up this protection, via `Optimize()`.