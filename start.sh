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

# Default settings
DEBUG_PORT=${DEBUG_PORT:-2345}
DEBUG_MODE=${DEBUG_MODE:-false}
BINARY_NAME="consent-server"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --debug)
            DEBUG_MODE=true
            shift
            ;;
        --debug-port)
            DEBUG_PORT="$2"
            shift 2
            ;;
        --help)
            echo "Consent Management Server Startup Script"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --debug              Enable debug mode with remote debugging"
            echo "  --debug-port PORT    Set debug port (default: 2345)"
            echo "  --help               Show this help message"
            echo ""
            echo "First-Time Setup:"
            echo "  Configuration is expected in: ./repository/conf/deployment.yaml"
            echo "  (relative to the binary location)"
            echo ""
            echo "  For development:"
            echo "    ./build.sh build    # From project root"
            echo "    cd bin && ./start.sh"
            echo ""
            echo "  For production:"
            echo "    Extract package to desired location"
            echo "    cd <extract-dir> && ./start.sh"
            echo ""
            echo "Configuration:"
            echo "  Server port and other settings are configured in:"
            echo "    ./repository/conf/deployment.yaml"
            echo ""
            echo "Examples:"
            echo "  $0                          Start server normally"
            echo "  $0 --debug                  Start in debug mode"
            echo "  $0 --debug --debug-port 3456  Start with custom debug port"
            echo ""
            echo "Remote Debugging:"
            echo "  1. Start server with --debug flag"
            echo "  2. Connect debugger to localhost:$DEBUG_PORT"
            echo "  3. VS Code: Use 'Attach to Remote' launch configuration"
            echo "  4. IntelliJ/GoLand: Create 'Go Remote' debug configuration"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

set -e  # Exit immediately if a command exits with a non-zero status

# Check for port conflicts
check_port() {
    local port=$1
    local port_name=$2
    if lsof -ti tcp:$port >/dev/null 2>&1; then
        echo ""
        echo "❌ Port $port is already in use"
        echo "   $port_name cannot start because another process is using port $port"
        echo ""
        echo "💡 To find the process using this port:"
        echo "   lsof -i tcp:$port"
        echo ""
        echo "💡 To stop the process:"
        echo "   kill -9 \$(lsof -ti tcp:$port)"
        echo ""
        exit 1
    fi
}

# Check if binary exists in current directory
if [ ! -f "./$BINARY_NAME" ]; then
    echo "❌ Binary not found: ./$BINARY_NAME"
    echo ""
    echo "💡 This script expects '$BINARY_NAME' in the same directory"
    echo ""
    echo "   For development: ./build.sh build && cd bin && ./start.sh"
    echo "   For production: Ensure this script is in the same directory as the binary"
    echo ""
    exit 1
fi

# Check if debug port is available (only when in debug mode)
if [ "$DEBUG_MODE" = "true" ]; then
    check_port $DEBUG_PORT "Debug server"
fi

# Check if Delve is available for debug mode
if [ "$DEBUG_MODE" = "true" ]; then
    # Check for dlv in PATH
    if ! command -v dlv &> /dev/null; then
        echo "❌ Debug mode requires Delve debugger"
        echo ""
        echo "💡 Install Delve using:"
        echo "   go install github.com/go-delve/delve/cmd/dlv@latest"
        echo ""
        echo "💡 Ensure GOPATH/bin is in your PATH:"
        echo "   export PATH=\$PATH:\$(go env GOPATH)/bin"
        echo ""
        echo "🔧 After installation, run: $0 --debug"
        exit 1
    fi
fi

# Run consent management server
if [ "$DEBUG_MODE" = "true" ]; then
    echo "🔒 Starting Consent Management Server in DEBUG mode..."
    echo "🐛 Remote debugger will listen on: localhost:$DEBUG_PORT"
    echo ""
    echo "💡 Connect using remote debugging configuration:"
    echo "   Host: 127.0.0.1, Port: $DEBUG_PORT"
    echo ""
    echo "📋 VS Code Launch Configuration (.vscode/launch.json):"
    echo '   {
     "name": "Attach to Remote",
     "type": "go",
     "request": "attach",
     "mode": "remote",
     "remotePath": "${workspaceFolder}",
     "port": '$DEBUG_PORT',
     "host": "127.0.0.1"
   }'
    echo ""

    # Set GIN_MODE to debug for verbose output when debugging
    export GIN_MODE=debug
    
    # Run with debugger
    dlv exec --listen=:$DEBUG_PORT --headless=true --api-version=2 --accept-multiclient --continue ./$BINARY_NAME &
    SERVER_PID=$!
else
    echo "🔒 Starting Consent Management Server..."
    echo ""

    # Run normally (GIN_MODE will default to release mode in main.go)
    ./$BINARY_NAME &
    SERVER_PID=$!
fi

# Cleanup function
cleanup() {
    echo -e "\n🛑 Stopping server..."
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
    fi
    # Kill any remaining dlv processes
    if [ "$DEBUG_MODE" = "true" ]; then
        pkill -f "dlv exec.*$BINARY_NAME" 2>/dev/null || true
    fi
}

# Cleanup on Ctrl+C and script exit
trap cleanup SIGINT SIGTERM EXIT

# Status
echo "🚀 Server running (PID: $SERVER_PID)"
echo "📋 Configuration: ./repository/conf/deployment.yaml"
echo ""
echo "Press Ctrl+C to stop the server."
echo ""

# Wait for background process
wait $SERVER_PID
