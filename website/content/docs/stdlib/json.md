---
title: "json"
description: "High-performance JSON parser, inspector, and builder powered by yyjson."
icon: "data_object"
weight: 360
---

```trux
import "json"
```

**What it is.** High-performance native JSON processing powered by `yyjson` and integrated directly into Trux's memory arenas. Supports fast document parsing, typed field queries, map conversions, and mutable document construction with zero external dependencies.

**Why.** HTTP servers and data pipelines frequently receive or emit JSON payloads. `yyjson` parses and serializes gigabytes per second while allocating directly into Trux's request and frame arenas.

**When you don't need it.** For tabular data, [`csv`](/docs/stdlib/csv/) or [`sqlite`](/docs/stdlib/sqlite/) may be more suitable.

---

## API Reference

### Parsing & Inspection

| Function | Returns | Description |
| :--- | :--- | :--- |
| `json.parse(input string)` | `(*json.Doc, error)` | Parses a JSON string into an immutable document. |
| `json.root(doc *json.Doc)` | `*json.Val` | Returns the root value of the document. |
| `json.get(val *json.Val, key string)` | `*json.Val` | Retrieves an object property by key (or `nil` if missing). |
| `json.at(val *json.Val, index int)` | `*json.Val` | Retrieves an array item by 0-based index (or `nil` if out of bounds). |
| `json.len(val *json.Val)` | `int` | Returns number of object members, array elements, or string length. |
| `json.valid(input string)` | `bool` | Validates JSON syntax without allocating a document. |

### Type Checking

| Function | Returns | Description |
| :--- | :--- | :--- |
| `json.isObject(val *json.Val)` | `bool` | Returns `true` if the value is a JSON object. |
| `json.isArray(val *json.Val)` | `bool` | Returns `true` if the value is a JSON array. |
| `json.isString(val *json.Val)` | `bool` | Returns `true` if the value is a string. |
| `json.isInt(val *json.Val)` | `bool` | Returns `true` if the value is an integer. |
| `json.isFloat(val *json.Val)` | `bool` | Returns `true` if the value is a floating-point number. |
| `json.isBool(val *json.Val)` | `bool` | Returns `true` if the value is a boolean. |
| `json.isNull(val *json.Val)` | `bool` | Returns `true` if the value is JSON `null` or `nil`. |

### Scalar Value & Field Extraction

| Function | Returns | Description |
| :--- | :--- | :--- |
| `json.getString(val *json.Val)` | `string` | Extracts string value (or `""` if not a string). |
| `json.getInt(val *json.Val)` | `int` | Extracts integer value (or `0` if not an integer). |
| `json.getFloat(val *json.Val)` | `float` | Extracts float value (or `0.0` if not a number). |
| `json.getBool(val *json.Val)` | `bool` | Extracts boolean value (or `false` if not a boolean). |
| `json.fieldString(val *json.Val, key string)` | `string` | Convenience: looks up `key` in object and extracts string. |
| `json.fieldInt(val *json.Val, key string)` | `int` | Convenience: looks up `key` in object and extracts int. |
| `json.fieldFloat(val *json.Val, key string)` | `float` | Convenience: looks up `key` in object and extracts float. |
| `json.fieldBool(val *json.Val, key string)` | `bool` | Convenience: looks up `key` in object and extracts bool. |

### Map Conversion & Document Serialization

| Function | Returns | Description |
| :--- | :--- | :--- |
| `json.decodeMap(input string)` | `(map[string]string, error)` | Decodes a top-level JSON object directly into a `map[string]string`. |
| `json.encodeMap(m map[string]string)` | `string` | Encodes a `map[string]string` into an escaped JSON object string. |
| `json.stringify(doc *json.Doc)` | `string` | Serializes an immutable document back into a JSON string. |

### Mutable Document Builder

| Function | Returns | Description |
| :--- | :--- | :--- |
| `json.newDoc()` | `*json.MutDoc` | Allocates a new mutable document in the current arena. |
| `json.newObject(doc *json.MutDoc)` | `*json.MutVal` | Creates a new mutable JSON object. |
| `json.newArray(doc *json.MutDoc)` | `*json.MutVal` | Creates a new mutable JSON array. |
| `json.setRoot(doc *json.MutDoc, root *json.MutVal)` | | Sets the root value of the document (statement only). |
| `json.addString(doc *json.MutDoc, obj *json.MutVal, key string, val string)` | | Adds a string key-value pair to an object (statement only). |
| `json.addInt(doc *json.MutDoc, obj *json.MutVal, key string, val int)` | | Adds an integer key-value pair to an object (statement only). |
| `json.addFloat(doc *json.MutDoc, obj *json.MutVal, key string, val float)` | | Adds a float key-value pair to an object (statement only). |
| `json.addBool(doc *json.MutDoc, obj *json.MutVal, key string, val bool)` | | Adds a boolean key-value pair to an object (statement only). |
| `json.addNull(doc *json.MutDoc, obj *json.MutVal, key string)` | | Adds a `null` key-value pair to an object (statement only). |
| `json.addVal(doc *json.MutDoc, obj *json.MutVal, key string, val *json.MutVal)` | | Attaches a nested value/object/array to an object (statement only). |
| `json.appendVal(arr *json.MutVal, val *json.MutVal)` | | Appends a value to an array (statement only). |
| `json.write(doc *json.MutDoc)` | `string` | Serializes a mutable document into a JSON string. |

---

## Examples

### Parsing Incoming JSON (HTTP Handler)

```trux
package main

import "http"
import "json"

func updateUser(request int) int {
  let doc *json.Doc, err error = json.parse(http.body(request))
  if err != nil {
    http.json(request, 400, "{\"error\":\"invalid json\"}\n")
    return 1
  }

  let root *json.Val = json.root(doc)
  let name string = json.fieldString(root, "name")
  let age int = json.fieldInt(root, "age")

  let m map[string]string = map[string]string{
    "status": "updated",
    "name": name,
  }
  http.json(request, 200, json.encodeMap(m))
  return 0
}
```
