#!/bin/bash
# Proto Development Setup Script
# Installs protoc, buf, and all necessary protoc plugins for development

set -e

echo "🔧 Setting up protobuf development environment..."

# Check if protoc is installed, if not install it
if ! command -v protoc >/dev/null 2>&1; then
    echo "📦 protoc is not installed. Installing via package manager..."
    if command -v brew >/dev/null 2>&1; then
        echo "Installing protoc via Homebrew..."
        brew install protobuf
    elif command -v apt-get >/dev/null 2>&1; then
        echo "Installing protoc via apt..."
        sudo apt-get update && sudo apt-get install -y protobuf-compiler
    elif command -v yum >/dev/null 2>&1; then
        echo "Installing protoc via yum..."
        sudo yum install -y protobuf-compiler
    elif command -v pacman >/dev/null 2>&1; then
        echo "Installing protoc via pacman..."
        sudo pacman -S protobuf
    else
        echo "❌ Could not detect package manager. Please install protoc manually:"
        echo "  macOS: brew install protobuf"
        echo "  Ubuntu/Debian: apt-get install protobuf-compiler"
        echo "  CentOS/RHEL: yum install protobuf-compiler"
        echo "  Arch: pacman -S protobuf"
        echo "  Or download from: https://github.com/protocolbuffers/protobuf/releases"
        exit 1
    fi
else
    echo "✅ protoc already installed: $(protoc --version)"
fi

# Check if buf is installed, if not install it
if ! command -v buf >/dev/null 2>&1; then
    echo "📦 buf is not installed. Installing..."
    if command -v brew >/dev/null 2>&1; then
        echo "Installing buf via Homebrew..."
        brew install bufbuild/buf/buf
    else
        echo "Installing buf via go install..."
        go install github.com/bufbuild/buf/cmd/buf@latest
    fi
else
    echo "✅ buf already installed: $(buf --version)"
fi

echo "📦 Installing Go protoc plugins..."
go install github.com/cosmos/gogoproto/protoc-gen-gocosmos@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo ""
echo "🎉 Protobuf development environment setup complete!"
echo ""
echo "Available tools:"
echo "  protoc: $(protoc --version)"
echo "  buf: $(buf --version)"
echo ""
echo "Installed protoc plugins:"
echo "  ✅ protoc-gen-gocosmos (Cosmos SDK)"
echo "  ✅ protoc-gen-grpc-gateway (gRPC Gateway)"
echo "  ✅ protoc-gen-openapiv2 (OpenAPI/Swagger)"
echo "  ✅ protoc-gen-go (Standard Go protobuf)"
echo "  ✅ protoc-gen-go-grpc (Go gRPC)"
echo ""
echo "You can now run:"
echo "  make proto-gen       # Generate Go files"
echo "  make proto-gen-docs  # Generate documentation"
echo "  make proto-gen-all   # Generate everything"
echo ""
