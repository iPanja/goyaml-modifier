package modifier

import (
  "reflect"
)

func buildMappingLookup(val reflect.Value, lookup map[string]reflect.Value) {
  switch val.Kind() {
  case reflect.Map:
    buildMapLookup(val, lookup)
  case reflect.Interface, reflect.Ptr:
    buildMappingLookup(val.Elem(), lookup)
    // TODO:
  case reflect.Struct:
    buildStructLookup(val, lookup)
  default:
    print("Unsupported type for `buildMappingLookup` ", val.Kind())
  }
}

func buildMapLookup(val reflect.Value, lookup map[string]reflect.Value) {
  iter := val.MapRange()
  for iter.Next() {
    mk := iter.Key()
    mv := iter.Value()

    if ShouldSkipField(mv) {
      continue
    }

    lookup[mk.String()] = mv
  }
}

func buildStructLookup(val reflect.Value, lookup map[string]reflect.Value) {
  t := val.Type()

  fields := reflect.VisibleFields(t)
  for _, field := range fields {
    v := val.FieldByName(field.Name)

    if IsInlineStructField(field) {
      buildMappingLookup(v, lookup)
      continue
    }

    if ShouldSkipField(v) || (IsOmitEmptyStructField(field) && v.IsZero()) {
      // TODO: set to nil so we can potentially remove node later?
      continue
    }

    k := getStructFieldKey(field)
    lookup[k] = v
  }
}

