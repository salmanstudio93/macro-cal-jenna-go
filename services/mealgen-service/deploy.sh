#!/bin/bash

# Deployment script for mealgen-service to Google Cloud Run
# This script handles environment variables and deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="mealgen-service"
PROJECT_ID="macro-path"
REGION="us-central1"

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

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check if environment variables are set (optional)
check_env_vars() {
    print_step "Checking environment variables..."
    
    if [ -z "$GEMINI_API_KEY" ]; then
        print_warning "GEMINI_API_KEY environment variable is not set"
        print_warning "You can set it later with: gcloud run services update ${SERVICE_NAME} --region=${REGION} --set-env-vars='GEMINI_API_KEY=your_key'"
    else
        print_status "GEMINI_API_KEY is set"
    fi
    
    if [ -z "$FOOD_API_KEY" ]; then
        print_warning "FOOD_API_KEY environment variable is not set"
        print_warning "You can set it later with: gcloud run services update ${SERVICE_NAME} --region=${REGION} --set-env-vars='FOOD_API_KEY=your_key'"
    else
        print_status "FOOD_API_KEY is set"
    fi
}

# Authenticate with Google Cloud
authenticate() {
    print_step "Authenticating with Google Cloud..."
    gcloud auth configure-docker --quiet
    gcloud config set project ${PROJECT_ID}
    print_status "Authentication complete"
}

# Build Docker image
build_image() {
    print_step "Building Docker image..."
    docker build -t gcr.io/${PROJECT_ID}/${SERVICE_NAME}:latest .
    print_status "Docker image built successfully"
}

# Push image to Google Container Registry
push_image() {
    print_step "Pushing image to Google Container Registry..."
    docker push gcr.io/${PROJECT_ID}/${SERVICE_NAME}:latest
    print_status "Image pushed successfully"
}

# Deploy to Cloud Run
deploy_service() {
    print_step "Deploying to Cloud Run..."
    
    gcloud run deploy ${SERVICE_NAME} \
        --image gcr.io/${PROJECT_ID}/${SERVICE_NAME}:latest \
        --platform managed \
        --region ${REGION} \
        --port 8080 \
        --memory 1Gi \
        --cpu 1 \
        --min-instances 0 \
        --max-instances 10 \
        --timeout 300 \
        --concurrency 80 \
        --allow-unauthenticated \
        --quiet
    
    print_status "Service deployed successfully"
}

# Set environment variables (only if they are set)
set_env_vars() {
    print_step "Setting environment variables..."
    
    if [ -n "$GEMINI_API_KEY" ] && [ -n "$FOOD_API_KEY" ]; then
        gcloud run services update ${SERVICE_NAME} \
            --region=${REGION} \
            --set-env-vars="GEMINI_API_KEY=${GEMINI_API_KEY},FOOD_API_KEY=${FOOD_API_KEY}" \
            --quiet
        print_status "Environment variables set successfully"
    elif [ -n "$GEMINI_API_KEY" ]; then
        gcloud run services update ${SERVICE_NAME} \
            --region=${REGION} \
            --set-env-vars="GEMINI_API_KEY=${GEMINI_API_KEY}" \
            --quiet
        print_status "GEMINI_API_KEY set successfully"
        print_warning "FOOD_API_KEY not set - you can set it later"
    elif [ -n "$FOOD_API_KEY" ]; then
        gcloud run services update ${SERVICE_NAME} \
            --region=${REGION} \
            --set-env-vars="FOOD_API_KEY=${FOOD_API_KEY}" \
            --quiet
        print_status "FOOD_API_KEY set successfully"
        print_warning "GEMINI_API_KEY not set - you can set it later"
    else
        print_warning "No environment variables set - you can set them later"
        print_warning "Use: ./deploy.sh env (after setting the variables)"
    fi
}

# Get service URL
get_service_url() {
    print_step "Getting service URL..."
    SERVICE_URL=$(gcloud run services describe ${SERVICE_NAME} --region=${REGION} --format="value(status.url)")
    print_status "Service URL: ${SERVICE_URL}"
    print_status "Health check: ${SERVICE_URL}/health"
}

# Test the deployed service
test_service() {
    print_step "Testing deployed service..."
    
    if [ -n "$SERVICE_URL" ]; then
        print_status "Testing health endpoint..."
        curl -f "${SERVICE_URL}/health" || print_warning "Health check failed"
    else
        print_warning "Service URL not available for testing"
    fi
}

# Show logs
show_logs() {
    print_step "Showing recent logs..."
    gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=${SERVICE_NAME}" --limit=20 --format="value(timestamp,textPayload)" --project=${PROJECT_ID}
}

# Main deployment function
deploy() {
    print_status "Starting Cloud Run deployment for ${SERVICE_NAME}..."
    
    check_env_vars
    authenticate
    build_image
    push_image
    deploy_service
    set_env_vars
    get_service_url
    test_service
    
    print_status "Deployment complete!"
    print_status "Service URL: ${SERVICE_URL}"
}

# Show help
show_help() {
    echo "Cloud Run Deployment Script for mealgen-service"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  deploy     - Full deployment (default)"
    echo "  build      - Build Docker image only"
    echo "  push       - Push image to registry only"
    echo "  logs       - Show service logs"
    echo "  test       - Test deployed service"
    echo "  env        - Set environment variables only"
    echo "  help       - Show this help message"
    echo ""
    echo "Required Environment Variables:"
    echo "  GEMINI_API_KEY  - API key for Gemini service"
    echo "  FOOD_API_KEY    - API key for Food service"
    echo ""
    echo "Example:"
    echo "  export GEMINI_API_KEY=your_key"
    echo "  export FOOD_API_KEY=your_key"
    echo "  $0 deploy"
}

# Main script logic
case "${1:-deploy}" in
    deploy)
        deploy
        ;;
    build)
        build_image
        ;;
    push)
        authenticate
        push_image
        ;;
    logs)
        show_logs
        ;;
    test)
        get_service_url
        test_service
        ;;
    env)
        check_env_vars
        set_env_vars
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
