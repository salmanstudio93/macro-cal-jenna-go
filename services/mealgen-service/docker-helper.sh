#!/bin/bash

# Docker helper script for mealgen-service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if .env file exists
check_env_file() {
    if [ ! -f .env ]; then
        print_warning ".env file not found. Creating from .env.example..."
        if [ -f .env.example ]; then
            cp .env.example .env
            print_status "Created .env file from .env.example"
            print_warning "Please edit .env file with your actual API keys before running the service"
        else
            print_error ".env.example file not found. Please create a .env file manually."
            exit 1
        fi
    fi
}

# Build Docker image
build_image() {
    print_status "Building Docker image..."
    docker build -t mealgen-service:latest .
    print_status "Docker image built successfully"
}

# Run with docker-compose
run_compose() {
    print_status "Starting service with docker-compose..."
    docker-compose up --build
}

# Run with Docker directly
run_docker() {
    print_status "Starting service with Docker..."
    docker run --env-file .env -p 8080:8080 mealgen-service:latest
}

# Stop containers
stop_containers() {
    print_status "Stopping containers..."
    docker-compose down
}

# Show logs
show_logs() {
    print_status "Showing logs..."
    docker-compose logs -f
}

# Clean up
cleanup() {
    print_status "Cleaning up Docker resources..."
    docker-compose down --volumes --remove-orphans
    docker system prune -f
}

# Show help
show_help() {
    echo "Docker Helper for mealgen-service"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  build     - Build Docker image"
    echo "  run       - Run with docker-compose"
    echo "  docker    - Run with Docker directly"
    echo "  stop      - Stop containers"
    echo "  logs      - Show logs"
    echo "  cleanup   - Clean up Docker resources"
    echo "  help      - Show this help message"
    echo ""
}

# Main script logic
case "${1:-help}" in
    build)
        check_env_file
        build_image
        ;;
    run)
        check_env_file
        run_compose
        ;;
    docker)
        check_env_file
        build_image
        run_docker
        ;;
    stop)
        stop_containers
        ;;
    logs)
        show_logs
        ;;
    cleanup)
        cleanup
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
