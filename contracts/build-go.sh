#!/bin/sh

# Script to build Go bindings from contract ABIs
# Run this from the contracts directory

PTH="./build-go"
rm -rf "$PTH"

# Create the output folder
mkdir -p "$PTH/validatorregistry"

# Generate Go bindings from ABI
abigen --abi=./abi/ValidatorRegistry.json --pkg=validatorregistry --out=./$PTH/validatorregistry/validatorregistry.go

# Get major version for Go module
MAJOR_VERSION=$(cut -d. -f1 VERSION 2>/dev/null || echo "1")

cd $PTH
go mod init github.com/Lumerin-protocol/contracts-go/v$MAJOR_VERSION
go mod tidy
cd ..

echo ""
echo "Success!"
echo "Go module initialized for version $MAJOR_VERSION"
