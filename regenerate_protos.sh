#!/bin/bash

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed."
    echo "Please install it using one of the following methods:"
    echo "- For Ubuntu/Debian: sudo apt-get install -y protobuf-compiler"
    echo "- For Alpine: apk add protobuf-dev"
    echo "- For Mac: brew install protobuf"
    echo ""
    echo "After installing protoc, also install protoc-gen-go with:"
    echo "go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go v1.31.0..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
fi

# Set the working directory to the project root
cd "$(dirname "$0")"
PROJECT_ROOT=$(pwd)

# Clean up previously generated files
echo "Cleaning up existing .pb.go files..."
find ./proto -name "*.pb.go" -type f -delete

# Find all proto files
PROTO_FILES=$(find ./proto -name "*.proto" -type f)

# Create a temporary directory structure that matches the import paths
TEMP_DIR=$(mktemp -d)
echo "Created temporary directory: $TEMP_DIR"

# Process each proto file
for proto_file in $PROTO_FILES; do
    echo "Processing $proto_file..."

    # Extract the directory name (e.g., waCommon from ./proto/waCommon/WACommon.proto)
    dir_name=$(dirname "$proto_file" | sed 's|./proto/||')

    # Create the directory in the temporary location
    mkdir -p "$TEMP_DIR/$dir_name"

    # Copy the proto file to the temporary location
    cp "$proto_file" "$TEMP_DIR/$dir_name/"
done

# Now compile all proto files using the temporary directory structure
for proto_file in $(find "$TEMP_DIR" -name "*.proto" -type f); do
    echo "Compiling $(basename "$proto_file")..."

    # Get the relative path for output
    rel_path=$(echo "$proto_file" | sed "s|$TEMP_DIR/||")
    dir_name=$(dirname "$rel_path")

    # Compile the proto file
    protoc -I="$TEMP_DIR" --go_out="$PROJECT_ROOT" --go_opt=module=github.com/shiestapoi/whatsmeow "$proto_file"
done

# Fix potential issues in ALL generated .pb.go files
echo "Applying fixes to all generated files..."
for pb_file in $(find ./proto -name "*.pb.go" -type f); do
    # Make a backup
    cp "$pb_file" "${pb_file}.bak"

    # Fix the slice bounds issue - covering all potential patterns
    sed -i 's/x\[-1\]/x[0]/g' "$pb_file"
    sed -i 's/dv\[-1\]/dv[0]/g' "$pb_file"
    sed -i 's/sv\[-1\]/sv[0]/g' "$pb_file"
    sed -i 's/bv\[-1\]/bv[0]/g' "$pb_file"

    echo "Fixed and backed up $pb_file"
done

# Clean up
echo "Removing temporary directory..."
rm -rf "$TEMP_DIR"

echo "Done regenerating protobuf files."
