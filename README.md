# sliceGen

A Go code generator that creates slice helper methods for struct types.

## Overview

sliceGen is a tool that automatically generates slice helper methods for Go struct types. It helps you avoid writing repetitive code for common slice operations.

## Features

- Generates slice helper methods for exported struct fields
- Supports multiple struct types in a single run
- Handles complex field types (arrays, slices, maps, channels, etc.)
- Option to ignore specific fields
- Supports nested packages
- Preserves package structure

## Installation
```bash
go install github.com/yourusername/sliceGen@latest
```

## Usage

```bash
sliceGen -type=StructName1,StructName2 -ignoreFields=Field1,Field2 -methods=filter,map,reduce,sort,groupby
```

### Parameters

- `-type`: Required. The struct name(s) to process. Can include package path and multiple struct names.
- `-ignoreFields`: Optional. Comma-separated list of field names to ignore.
- `-methods`: Optional. Extra helper methods to generate. Available methods:
  - `filter`: Generate Filter method
  - `map`: Generate Map method
  - `reduce`: Generate Reduce method
  - `sort`: Generate Sort method
  - `groupby`: Generate GroupBy method

### Example

Given a struct:
```go
type Book struct {
    Name string
    Price float64
    author string // unexported field
}
```

Running:
```bash
sliceGen -type Book
```

Will generate:
```go
type BookSlice []Book
func (s BookSlice) Name() []string {
    return lo.Map(s, func(item Book, index int) string {
        return item.Name
    })
}

func (s BookSlice) Price() []float64 {
    return lo.Map(s, func(item Book, index int) float64 {
        return item.Price
    })
}
```

## Test
```bash
go test -v -cover
```
## Supported Types

- Basic types (string, int, etc.)
- Pointers
- Arrays and Slices
- Maps
- Channels
- Interfaces
- Functions
- Nested structs
- Complex combinations of above types

## Dependencies

- github.com/samber/lo

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.