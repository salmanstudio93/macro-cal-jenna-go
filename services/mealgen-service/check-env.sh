#!/bin/bash

# Environment variable checker for mealgen-service deployment

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

# Check if environment variables are set
check_env_vars() {
    print_step "Checking environment variables..."
    
    local missing_vars=()
    local all_good=true
    
    # Check GEMINI_API_KEY
    if [ -z "$GEMINI_API_KEY" ]; then
        print_error "GEMINI_API_KEY is not set"
        missing_vars+=("GEMINI_API_KEY")
        all_good=false
    else
        print_status "GEMINI_API_KEY: [SET]"
    fi
    
    # Check FOOD_API_KEY
    if [ -z "$FOOD_API_KEY" ]; then
        print_error "FOOD_API_KEY is not set"
        missing_vars+=("FOOD_API_KEY")
        all_good=false
    else
        print_status "FOOD_API_KEY: [SET]"
    fi
    
    # Check PORT (optional)
    if [ -n "$PORT" ]; then
        print_status "PORT: $PORT"
    else
        print_status "PORT: [NOT SET] (will use default 8080)"
    fi
    
    # Check LOG_LEVEL (optional)
    if [ -n "$LOG_LEVEL" ]; then
        print_status "LOG_LEVEL: $LOG_LEVEL"
    else
        print_status "LOG_LEVEL: [NOT SET] (will use default 'info')"
    fi
    
    if [ "$all_good" = true ]; then
        print_status "All required environment variables are set!"
        return 0
    else
        print_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        return 1
    fi
}

# Show how to set environment variables
show_setup_instructions() {
    print_step "How to set environment variables:"
    echo ""
    echo "1. Set them in your current shell:"
    echo "   export GEMINI_API_KEY=\"your_gemini_api_key_here\""
    echo "   export FOOD_API_KEY=\"your_food_api_key_here\""
    echo ""
    echo "2. Or add them to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "   echo 'export GEMINI_API_KEY=\"your_key\"' >> ~/.bashrc"
    echo "   echo 'export FOOD_API_KEY=\"your_key\"' >> ~/.bashrc"
    echo "   source ~/.bashrc"
    echo ""
    echo "3. Or create a .env file and source it:"
    echo "   echo 'GEMINI_API_KEY=your_key' > .env"
    echo "   echo 'FOOD_API_KEY=your_key' >> .env"
    echo "   source .env"
    echo ""
}

# Test deployment readiness
test_deployment() {
    print_step "Testing deployment readiness..."
    
    if check_env_vars; then
        print_status "✅ Environment variables are ready for deployment"
        print_status "You can now run: make deploy SERVICE=mealgen-service"
        return 0
    else
        print_warning "❌ Environment variables are not ready"
        show_setup_instructions
        return 1
    fi
}

# Show current environment status
show_status() {
    print_step "Current environment status:"
    echo ""
    echo "GEMINI_API_KEY: ${GEMINI_API_KEY:+[SET]}${GEMINI_API_KEY:-[NOT SET]}"
    echo "FOOD_API_KEY: ${FOOD_API_KEY:+[SET]}${FOOD_API_KEY:-[NOT SET]}"
    echo "PORT: ${PORT:-[NOT SET - will use default 8080]}"
    echo "LOG_LEVEL: ${LOG_LEVEL:-[NOT SET - will use default 'info']}"
    echo ""
}

# Main function
main() {
    echo "🔍 Environment Variable Checker for mealgen-service"
    echo ""
    
    case "${1:-check}" in
        check)
            check_env_vars
            ;;
        test)
            test_deployment
            ;;
        status)
            show_status
            ;;
        help|--help|-h)
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  check   - Check if environment variables are set (default)"
            echo "  test    - Test deployment readiness"
            echo "  status  - Show current environment status"
            echo "  help    - Show this help message"
            ;;
        *)
            print_error "Unknown command: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"
