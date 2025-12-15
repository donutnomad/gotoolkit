# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains Go code generation tools:

1. **sliceGen** - Generates slice helper methods for Go struct types
2. **approveGen** - Generates approval workflow code based on annotations
3. **swagGen** - Generates Swagger documentation and Gin binding code from interface definitions

## Commands

### Build Commands
```bash
# Build approveGen tool
make buildApprove

# Build swagGen tool
make buildSwag
```

### Test Commands
```bash
# Run tests (if available)
go test -v -cover
```

## Code Architecture

### sliceGen
- Located in `sliceGen/` directory
- Generates helper methods for struct slice operations
- Uses `github.com/samber/lo` for functional programming utilities
- Parses Go AST to analyze struct definitions

### approveGen
- Located in `approveGen/` directory
- Processes `@Approve` annotations in Go code
- Generates method approval workflow code
- Uses `github.com/dave/jennifer/jen` for code generation

### swagGen
- Located in `swagGen/` directory
- Parses interface definitions with Swagger annotations
- Generates both Swagger documentation and Gin binding code
- Key components:
  - Interface parser (`interface_parser.go`)
  - Swagger generator (`swagger_generator.go`)
  - Gin generator (`gin_generator.go`)
  - Import manager (`import_manager.go`)

## Dependencies

- Go 1.23.1
- github.com/samber/lo (functional programming utilities)
- github.com/dave/jennifer (code generation)
- golang.org/x/tools (Go tools and AST manipulation)
- github.com/bytedance/sonic (JSON processing)
- github.com/Xuanwo/gg (code generation)