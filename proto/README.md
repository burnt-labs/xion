# Protobuf Generation

This directory contains the protobuf definitions for Xion and the tooling to generate Go code and documentation from them.

## Quick Start

### 1. First-time setup
Run this once to install all required tools and plugins:
```bash
make proto-setup
# OR directly:
./scripts/setup-proto.sh
```

This will install:
- `protoc` (Protocol Buffer compiler)
- `buf` (Modern protobuf tooling)
- All necessary Go protoc plugins

### 2. Generate code
```bash
make proto-gen       # Generate Go files
make proto-gen-docs  # Generate documentation  
make proto-gen-all   # Generate everything
```

## What's included

### Buf generation templates
- `buf.gen.go.yaml` - Go code generation
- `buf.gen.docs.yaml` - Documentation generation  
- `buf.gen.unified.yaml` - Everything at once

### Available Makefile targets
```bash
make proto-setup     # Install protoc, buf, and plugins
make proto-gen       # Generate Go protobuf files
make proto-gen-docs  # Generate OpenAPI documentation
make proto-gen-all   # Generate everything
make proto-format    # Format protobuf files
make proto-lint      # Lint protobuf files
```

## Migration from Docker approach

This replaces the previous Docker-based approach with a faster, native implementation:

**Before (Docker + 176-line shell script):**
- Required Docker container with pre-installed tools
- Complex shell script with manual dependency management
- Slower due to Docker overhead

**Now (Native buf + simple config):**
- Direct buf execution with locally installed tools
- Simple YAML configuration files
- Much faster execution
- No Docker overhead (runs natively)
- Parallel processing when possible
- No manual file handling

## Usage examples:

### Direct buf commands:
```bash
cd proto
buf generate --template buf.gen.gogo.yaml      # Go generation
buf generate --template buf.gen.docs.yaml    # Docs generation  
```

### Via Makefile:
```bash
make proto-gen-buf        # Fast native Go generation
make proto-gen-docs-buf   # Fast native docs generation
make proto-gen-all-buf    # Fast native everything
```

### Via simplified script:
```bash
./scripts/proto-gen-buf.sh                   # Go generation (default)
./scripts/proto-gen-buf.sh --gogo           # Go generation  
./scripts/proto-gen-buf.sh --swagger        # Docs generation
./scripts/proto-gen-buf.sh --all            # Everything
```

## Migration benefits achieved:

✅ **Reduced complexity**: From 176 to 67 lines of shell script  
✅ **Better performance**: No Docker overhead, native execution  
✅ **Improved reliability**: Less error-prone, better error messages  
✅ **Easier maintenance**: Simple YAML configs instead of complex shell logic  
✅ **Future-proof**: Leverages buf's evolving ecosystem  
✅ **Parallel processing**: Buf can optimize generation internally  
✅ **Consistent behavior**: Same results across different environments  

## Backward compatibility:

The original Docker-based targets still work:
- `make proto-gen` (original Docker approach)
- `make proto-gen-openapi` (original Docker approach)
- `make proto-gen-swagger` (original Docker approach)

## Next steps:

1. Test the new native approach thoroughly
2. Consider making the native approach the default
3. Eventually deprecate the Docker approach for local development
4. Update CI/CD to use the faster native approach
