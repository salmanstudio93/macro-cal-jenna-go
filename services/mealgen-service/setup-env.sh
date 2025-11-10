#!/bin/bash

# Environment setup script for mealgen-service
# This script helps set up environment variables for deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check if .env file exists
check_env_file() {
    if [ -f .env ]; then
        print_status ".env file found"
        return 0
    else
        print_warning ".env file not found"
        return 1
    fi
}

# Load environment variables from .env file
load_env() {
    if check_env_file; then
        print_step "Loading environment variables from .env file..."
        export $(cat .env | grep -v "^#" | xargs)
        print_status "Environment variables loaded"
    else
        print_warning "No .env file found. Please set environment variables manually:"
        echo "  export GEMINI_API_KEY=your_key_here"
        echo "  export FOOD_API_KEY=your_key_here"
    fi
}

# Create .env file from template
create_env_file() {
    if [ -f .env.example ]; then
        print_step "Creating .env file from .env.example..."
        cp .env.example .env
        print_status ".env file created"
        print_warning "Please edit .env file with your actual API keys"
    else
        print_error ".env.example file not found"
        exit 1
    fi
}

# Validate required environment variables
validate_env() {
    print_step "Validating environment variables..."
    
    local missing_vars=()
    
    if [ -z "$GEMINI_API_KEY" ]; then
        missing_vars+=("GEMINI_API_KEY")
    fi
    
    if [ -z "$FOOD_API_KEY" ]; then
        missing_vars+=("FOOD_API_KEY")
    fi
    
    if [ ${#missing_vars[@]} -eq 0 ]; then
        print_status "All required environment variables are set"
        return 0
    else
        print_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        return 1
    fi
}

# Show current environment status
show_status() {
    print_step "Current environment status:"
    echo ""
    echo "GEMINI_API_KEY: ${GEMINI_API_KEY:+[SET]}${GEMINI_API_KEY:-[NOT SET]}"
    echo "FOOD_API_KEY: ${FOOD_API_KEY:+[SET]}${FOOD_API_KEY:-[NOT SET]}"
    echo "PORT: ${PORT:-8080}"
    echo "LOG_LEVEL: ${LOG_LEVEL:-info}"
    echo ""
}

# Interactive setup
interactive_setup() {
    print_step "Interactive environment setup..."
    
    if [ -z "$GEMINI_API_KEY" ]; then
        read -p "Enter GEMINI_API_KEY: " GEMINI_API_KEY
        export GEMINI_API_KEY
    fi
    
    if [ -z "$FOOD_API_KEY" ]; then
        read -p "Enter FOOD_API_KEY: " FOOD_API_KEY
        export FOOD_API_KEY
    fi
    
    print_status "Environment variables set interactively"
}

# Main setup function
setup() {
    print_status "Setting up environment for mealgen-service..."
    
    # Try to load from .env file first
    load_env
    
    # If no .env file, try interactive setup
    if ! validate_env; then
        print_warning "Required environment variables not found"
        echo ""
        echo "Choose an option:"
        echo "1. Create .env file from template"
        echo "2. Set variables interactively"
        echo "3. Exit and set variables manually"
        echo ""
        read -p "Enter your choice (1-3): " choice
        
        case $choice in
            1)
                create_env_file
                print_warning "Please edit .env file and run this script again"
                ;;
            2)
                interactive_setup
                ;;
            3)
                print_warning "Please set environment variables manually and run again"
                exit 1
                ;;
            *)
                print_error "Invalid choice"
                exit 1
                ;;
        esac
    fi
    
    # Final validation
    if validate_env; then
        print_status "Environment setup complete!"
        show_status
    else
        print_error "Environment setup failed"
        exit 1
    fi
}

# Show help
show_help() {
    echo "Environment Setup Script for mealgen-service"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  setup      - Interactive environment setup (default)"
    echo "  load       - Load environment from .env file"
    echo "  validate   - Validate current environment"
    echo "  status     - Show current environment status"
    echo "  create     - Create .env file from template"
    echo "  help       - Show this help message"
    echo ""
    echo "Required Environment Variables:"
    echo "  GEMINI_API_KEY  - API key for Gemini service"
    echo "  FOOD_API_KEY    - API key for Food service"
    echo ""
    echo "Example:"
    echo "  $0 setup        # Interactive setup"
    echo "  $0 load         # Load from .env file"
    echo "  $0 validate     # Check if variables are set"
}

# Main script logic
case "${1:-setup}" in
    setup)
        setup
        ;;
    load)
        load_env
        validate_env
        ;;
    validate)
        validate_env
        ;;
    status)
        show_status
        ;;
    create)
        create_env_file
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
