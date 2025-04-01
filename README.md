# Go YAML Modifier (MIMO)

A minimally invasive YAML modifier.

## Purpsoe

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
```

### Here are possible outputs that could be generated based on the input

#### Example 1

```yaml
data:
  var: &name Somebody
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

```

```
