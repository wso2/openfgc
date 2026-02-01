#!/bin/bash

# ----------------------------------------------------------------------------
# Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.
# ----------------------------------------------------------------------------

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# --- Set Default OS and architecture --- 
# Auto-detect GO OS
DEFAULT_OS=$(go env GOOS 2>/dev/null)
if [ -z "$DEFAULT_OS" ]; then
  UNAME_OS="$(uname -s)"
  case "$UNAME_OS" in
    Darwin) DEFAULT_OS="darwin" ;;
    Linux) DEFAULT_OS="linux" ;;
    MINGW*|MSYS*|CYGWIN*) DEFAULT_OS="windows" ;;
    *) echo "Unsupported OS: $UNAME_OS"; exit 1 ;;
  esac
fi

# Auto-detect GO ARCH
DEFAULT_ARCH=$(go env GOARCH 2>/dev/null)
if [ -z "$DEFAULT_ARCH" ]; then
  UNAME_ARCH="$(uname -m)"
  case "$UNAME_ARCH" in
    x86_64|amd64) DEFAULT_ARCH="amd64" ;;
    arm64|aarch64) DEFAULT_ARCH="arm64" ;;
    *) echo "Unsupported architecture: $UNAME_ARCH"; exit 1 ;;
  esac
fi

GO_OS=${2:-$DEFAULT_OS}
GO_ARCH=${3:-$DEFAULT_ARCH}

echo "================================================================"
echo "Using GO OS: $GO_OS and ARCH: $GO_ARCH"
echo "================================================================"

# Version management
VERSION_FILE="version.txt"
if [ -f "$VERSION_FILE" ]; then
    VERSION=$(cat "$VERSION_FILE")
else
    VERSION="1.0.0"
fi

# Configuration
BINARY_NAME="consent-server"
OUTPUT_DIR="bin"
SOURCE_DIR="consent-server/cmd/server"
CONFIG_SOURCE="consent-server/cmd/server/repository/conf/deployment.yaml"
TARGET_DIR="target"
DIST_DIR="$TARGET_DIR/dist"

# Package naming
PACKAGE_OS=$GO_OS
PACKAGE_ARCH=$GO_ARCH

# Normalize OS name for distribution packaging
if [ "$GO_OS" = "darwin" ]; then
    PACKAGE_OS=macos
elif [ "$GO_OS" = "windows" ]; then
    PACKAGE_OS="win"
fi

if [ "$GO_ARCH" = "amd64" ]; then
    PACKAGE_ARCH=x64
fi

PRODUCT_FOLDER="${BINARY_NAME}-${VERSION}-${PACKAGE_OS}-${PACKAGE_ARCH}"

# ============================================================================
# Functions
# ============================================================================

function clean_all() {
    echo "================================================================"
    echo "Cleaning all build artifacts..."
    rm -rf "$TARGET_DIR"
    rm -rf "$OUTPUT_DIR"
    echo "✓ All build artifacts cleaned"
    echo "================================================================"
}

function clean() {
    echo "================================================================"
    echo "Cleaning build artifacts..."
    rm -rf "$OUTPUT_DIR"
    echo "✓ Build artifacts cleaned"
    echo "================================================================"
}

function build_binary() {
    echo "================================================================"
    echo "Building Consent Management API Server..."
    
    # Set binary name with .exe extension for Windows
    local output_binary="$BINARY_NAME"
    if [ "$GO_OS" = "windows" ]; then
        output_binary="${BINARY_NAME}.exe"
    fi
    
    # Clean previous build
    if [ -d "$OUTPUT_DIR" ]; then
        echo "Cleaning previous build..."
        rm -rf "$OUTPUT_DIR"
    fi
    
    # Create directory structure
    echo "Creating directory structure..."
    mkdir -p "$OUTPUT_DIR/repository/conf"
    
    # Build the binary with version and build date
    echo "Compiling binary for $GO_OS/$GO_ARCH..."
    cd consent-server
    GOOS=$GO_OS GOARCH=$GO_ARCH CGO_ENABLED=0 go build \
        -ldflags "-X 'main.version=$VERSION' -X 'main.buildDate=$(date -u '+%Y-%m-%d %H:%M:%S UTC')'" \
        -o "../$OUTPUT_DIR/$output_binary" "./cmd/server"
    cd ..
    
    # Copy configuration
    echo "Copying configuration..."
    cp "$CONFIG_SOURCE" "$OUTPUT_DIR/repository/conf/deployment.yaml"
    
    # Copy start script
    echo "Copying start script..."
    cp start.sh "$OUTPUT_DIR/start.sh"
    chmod +x "$OUTPUT_DIR/start.sh"
    
    # Copy database scripts
    if [ -d "consent-server/dbscripts" ]; then
        echo "Copying database scripts..."
        mkdir -p "$OUTPUT_DIR/dbscripts"
        cp consent-server/dbscripts/*.sql "$OUTPUT_DIR/dbscripts/" 2>/dev/null || true
    fi
    
    # Copy API specifications
    if [ -d "api" ]; then
        echo "Copying API specifications..."
        mkdir -p "$OUTPUT_DIR/api"
        cp api/*.yaml "$OUTPUT_DIR/api/" 2>/dev/null || true
    fi
    
    # Make binary executable (not needed for Windows)
    if [ "$GO_OS" != "windows" ]; then
        chmod +x "$OUTPUT_DIR/$output_binary"
    fi
    
    echo ""
    echo "✓ Build completed successfully!"
    echo ""
    echo "Build output:"
    echo "  Binary: $OUTPUT_DIR/$output_binary"
    echo "  Start Script: $OUTPUT_DIR/start.sh"
    echo "  Config: $OUTPUT_DIR/repository/conf/deployment.yaml"
    if [ -d "$OUTPUT_DIR/dbscripts" ]; then
        echo "  DB Scripts: $OUTPUT_DIR/dbscripts/"
    fi
    if [ -d "$OUTPUT_DIR/api" ]; then
        echo "  API Specs: $OUTPUT_DIR/api/"
    fi
    echo ""
    echo "To run the server:"
    echo "  cd $OUTPUT_DIR && ./start.sh"
    echo ""
    echo "Or with debug mode:"
    echo "  cd $OUTPUT_DIR && ./start.sh --debug"
    echo ""
    echo "================================================================"
}

function package() {
    echo "================================================================"
    echo "Creating distribution package..."
    
    # Build first
    build_binary
    
    # Create distribution directory
    mkdir -p "$DIST_DIR/$PRODUCT_FOLDER"
    
    # Copy everything from bin to dist
    echo "Copying build artifacts to distribution..."
    cp -r "$OUTPUT_DIR/"* "$DIST_DIR/$PRODUCT_FOLDER/"
    
    # Copy version file
    if [ -f "$VERSION_FILE" ]; then
        cp "$VERSION_FILE" "$DIST_DIR/$PRODUCT_FOLDER/"
    fi
    
    # Copy README if exists
    if [ -f "README.md" ]; then
        cp "README.md" "$DIST_DIR/$PRODUCT_FOLDER/"
    fi
    
    # Copy LICENSE if exists
    if [ -f "LICENSE" ]; then
        cp "LICENSE" "$DIST_DIR/$PRODUCT_FOLDER/"
    fi
    
    # Create zip file
    echo "Creating zip archive..."
    (cd "$DIST_DIR" && zip -r "$PRODUCT_FOLDER.zip" "$PRODUCT_FOLDER")
    
    # Clean up unzipped folder
    rm -rf "$DIST_DIR/$PRODUCT_FOLDER"
    
    echo ""
    echo "✓ Distribution package created successfully!"
    echo ""
    echo "Package: $DIST_DIR/$PRODUCT_FOLDER.zip"
    echo ""
    echo "================================================================"
}

function run_server() {
    echo "================================================================"
    echo "Running Consent Management API Server..."
    
    # Build first if binary doesn't exist
    if [ ! -f "$OUTPUT_DIR/$BINARY_NAME" ]; then
        echo "Binary not found. Building first..."
        build_binary
    fi
    
    echo "Starting server..."
    cd "$OUTPUT_DIR" && "./$BINARY_NAME"
    echo "================================================================"
}

function test_unit() {
    echo "================================================================"
    echo "Running unit tests..."
    cd consent-server || exit 1
    go test ./internal/... -v -cover
    cd "$SCRIPT_DIR" || exit 1
    echo "================================================================"
}

function test_integration() {
    echo "================================================================"
    echo "Running integration tests..."
    
    # Build the server first if binary doesn't exist
    if [ ! -f "$OUTPUT_DIR/$BINARY_NAME" ]; then
        echo "Binary not found. Building first..."
        build_binary
    fi
    
    # Replace app config with test config for integration tests
    echo "Copying test configuration..."
    if [ -f "tests/integration/repository/conf/deployment.yaml" ]; then
        cp tests/integration/repository/conf/deployment.yaml "$OUTPUT_DIR/repository/conf/deployment.yaml"
        echo "✓ Test configuration copied"
    else
        echo "⚠ Warning: Test configuration not found, using default config"
    fi
    
    # Run integration test suite
    echo "Starting integration test suite..."
    cd tests/integration || exit 1
    go run main.go
    TEST_EXIT_CODE=$?
    cd "$SCRIPT_DIR" || exit 1
    
    if [ $TEST_EXIT_CODE -ne 0 ]; then
        echo "✗ Integration tests failed"
        exit 1
    fi
    
    echo "✓ Integration tests passed"
    echo "================================================================"
}

function test_all() {
    test_unit
    test_integration
}

function show_help() {
    echo "Consent Management API Build Script"
    echo ""
    echo "Usage: ./build.sh {command} [OS] [ARCH]"
    echo ""
    echo "Commands:"
    echo "  clean            - Clean build artifacts"
    echo "  clean_all        - Clean all artifacts including distributions"
    echo "  build            - Build the binary and prepare output directory"
    echo "  package          - Build and create distribution package (zip)"
    echo "  run              - Build and run the server"
    echo "  test_unit        - Run unit tests"
    echo "  test_integration - Run integration tests"
    echo "  test             - Run all tests"
    echo "  help             - Show this help message"
    echo ""
    echo "Optional Arguments:"
    echo "  OS               - Target operating system (darwin, linux, windows)"
    echo "                     Default: auto-detected ($DEFAULT_OS)"
    echo "  ARCH             - Target architecture (amd64, arm64)"
    echo "                     Default: auto-detected ($DEFAULT_ARCH)"
    echo ""
    echo "Examples:"
    echo "  ./build.sh build                    # Build for current platform"
    echo "  ./build.sh build linux amd64        # Build for Linux AMD64"
    echo "  ./build.sh build darwin arm64       # Build for macOS ARM64"
    echo "  ./build.sh package                  # Create distribution package"
    echo "  ./build.sh run                      # Build and run server"
    echo ""
}

# ============================================================================
# Main script execution
# ============================================================================

case "$1" in
    clean)
        clean
        ;;
    clean_all)
        clean_all
        ;;
    build)
        build_binary
        ;;
    package)
        package
        ;;
    run)
        run_server
        ;;
    test_unit)
        test_unit
        ;;
    test_integration)
        test_integration
        ;;
    test)
        test_all
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Error: Unknown command '$1'"
        echo ""
        show_help
        exit 1
        ;;
esac
