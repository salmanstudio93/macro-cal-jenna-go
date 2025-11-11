# Quick Deploy Script for Reset Fitness App - Meal Generation Service
# This script deploys the mealgen-service to Google Cloud Run

$PROJECT_ID = "reset-fitness-app"
$SERVICE_NAME = "mealgen-service"
$REGION = "us-central1"
$IMAGE_NAME = "gcr.io/$PROJECT_ID/$SERVICE_NAME:latest"

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Reset Fitness App - Cloud Deployment" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Project ID: $PROJECT_ID" -ForegroundColor Yellow
Write-Host "Service: $SERVICE_NAME" -ForegroundColor Yellow
Write-Host "Region: $REGION" -ForegroundColor Yellow
Write-Host ""

# Navigate to project root
$ProjectRoot = "D:\Studio93\macro calculator web\macro-path-backend-mealgen-service"
cd $ProjectRoot
Write-Host "Working directory: $ProjectRoot" -ForegroundColor Gray
Write-Host ""

# Step 1: Configure GCP project
Write-Host "Step 1/6: Configuring GCP project..." -ForegroundColor Cyan
gcloud config set project $PROJECT_ID
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to set project. Please ensure you're logged in with 'gcloud auth login'" -ForegroundColor Red
    exit 1
}
Write-Host "  Project configured" -ForegroundColor Green
Write-Host ""

# Step 2: Enable required APIs
Write-Host "Step 2/6: Enabling required APIs..." -ForegroundColor Cyan
gcloud services enable run.googleapis.com containerregistry.googleapis.com --project=$PROJECT_ID --quiet
Write-Host "  APIs enabled" -ForegroundColor Green
Write-Host ""

# Step 3: Authenticate Docker
Write-Host "Step 3/6: Authenticating Docker with GCP..." -ForegroundColor Cyan
gcloud auth configure-docker --quiet
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to authenticate Docker" -ForegroundColor Red
    exit 1
}
Write-Host "  Docker authenticated" -ForegroundColor Green
Write-Host ""

# Step 4: Build Docker image
Write-Host "Step 4/6: Building Docker image..." -ForegroundColor Cyan
Write-Host "  This may take a few minutes..." -ForegroundColor Gray
docker build -t $IMAGE_NAME "services\$SERVICE_NAME"
if ($LASTEXITCODE -ne 0) {
    Write-Host "Docker build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "  Docker image built successfully" -ForegroundColor Green
Write-Host ""

# Step 5: Push to Container Registry
Write-Host "Step 5/6: Pushing image to Google Container Registry..." -ForegroundColor Cyan
Write-Host "  Uploading image..." -ForegroundColor Gray
docker push $IMAGE_NAME
if ($LASTEXITCODE -ne 0) {
    Write-Host "Docker push failed!" -ForegroundColor Red
    exit 1
}
Write-Host "  Image pushed successfully" -ForegroundColor Green
Write-Host ""

# Step 6: Deploy to Cloud Run
Write-Host "Step 6/6: Deploying to Cloud Run..." -ForegroundColor Cyan
Write-Host "  Creating/Updating service..." -ForegroundColor Gray
gcloud run deploy $SERVICE_NAME `
    --image $IMAGE_NAME `
    --platform managed `
    --region $REGION `
    --project $PROJECT_ID `
    --port 8080 `
    --memory 1Gi `
    --cpu 1 `
    --min-instances 0 `
    --max-instances 10 `
    --timeout 300 `
    --concurrency 80 `
    --allow-unauthenticated `
    --set-env-vars GEMINI_API_KEY="AIzaSyD7C2E3682Avm36w7OynNP_4j9DaX8tXTw",FOOD_API_KEY="83b88a97d33c889f3fa582d1d190d53b2655a72da9353465f70fbaaa18d498a0" `
    --quiet

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "  DEPLOYMENT SUCCESSFUL!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    
    # Get service URL
    Write-Host "Fetching service URL..." -ForegroundColor Cyan
    $SERVICE_URL = gcloud run services describe $SERVICE_NAME --region=$REGION --project=$PROJECT_ID --format="value(status.url)"
    
    Write-Host ""
    Write-Host "Your service is live at:" -ForegroundColor Green
    Write-Host $SERVICE_URL -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Available endpoints:" -ForegroundColor Cyan
    Write-Host "  GET  $SERVICE_URL/health" -ForegroundColor White
    Write-Host "  GET  $SERVICE_URL/" -ForegroundColor White
    Write-Host "  POST $SERVICE_URL/" -ForegroundColor White
    Write-Host "  POST $SERVICE_URL/regenerate" -ForegroundColor White
    Write-Host "  POST $SERVICE_URL/program/generate-program" -ForegroundColor White
    Write-Host ""
    Write-Host "Test health check:" -ForegroundColor Cyan
    Write-Host "  Invoke-RestMethod -Uri $SERVICE_URL/health" -ForegroundColor Gray
    Write-Host ""
} else {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "  DEPLOYMENT FAILED!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Check the logs with:" -ForegroundColor Yellow
    Write-Host "  gcloud run services logs read $SERVICE_NAME --region=$REGION --project=$PROJECT_ID" -ForegroundColor Gray
    exit 1
}

