# Build locally and push to Docker Hub (No gcloud needed!)
# Then deploy via Cloud Console

param(
    [Parameter(Mandatory=$false)]
    [string]$DockerUsername = "",
    
    [Parameter(Mandatory=$false)]
    [string]$Version = "latest"
)

$SERVICE_NAME = "mealgen-service"
$LOCAL_TAG = "$SERVICE_NAME:local"

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Docker Build & Push for Cloud Run" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Navigate to project root
$ProjectRoot = "D:\Studio93\macro calculator web\macro-path-backend-mealgen-service"
cd $ProjectRoot

# Step 1: Build locally
Write-Host "Step 1/4: Building Docker image locally..." -ForegroundColor Cyan
docker build -t $LOCAL_TAG "services\$SERVICE_NAME"

if ($LASTEXITCODE -ne 0) {
    Write-Host "Docker build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "  Build successful!" -ForegroundColor Green
Write-Host ""

# Step 2: Test locally (optional, can skip)
Write-Host "Step 2/4: Local image ready for testing" -ForegroundColor Cyan
Write-Host "  To test locally, run:" -ForegroundColor Yellow
Write-Host "    docker run -p 8080:8080 -e GEMINI_API_KEY=`"YOUR_KEY`" -e FOOD_API_KEY=`"YOUR_KEY`" $LOCAL_TAG" -ForegroundColor Gray
Write-Host "  Then test: Invoke-RestMethod -Uri http://localhost:8080/health" -ForegroundColor Gray
Write-Host ""

# Step 3: Tag and push
if ($DockerUsername -eq "") {
    Write-Host "Step 3/4: Ready to push to registry" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Choose your deployment method:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Option 1: Push to Docker Hub (Easiest - No gcloud needed)" -ForegroundColor Green
    Write-Host "  1. docker login" -ForegroundColor White
    Write-Host "  2. docker tag $LOCAL_TAG yourusername/$SERVICE_NAME`:$Version" -ForegroundColor White
    Write-Host "  3. docker push yourusername/$SERVICE_NAME`:$Version" -ForegroundColor White
    Write-Host ""
    Write-Host "Option 2: Push to Google Container Registry" -ForegroundColor Green
    Write-Host "  1. gcloud auth configure-docker" -ForegroundColor White
    Write-Host "  2. docker tag $LOCAL_TAG gcr.io/reset-fitness-app/$SERVICE_NAME`:$Version" -ForegroundColor White
    Write-Host "  3. docker push gcr.io/reset-fitness-app/$SERVICE_NAME`:$Version" -ForegroundColor White
    Write-Host ""
} else {
    Write-Host "Step 3/4: Tagging for Docker Hub..." -ForegroundColor Cyan
    $DOCKER_TAG = "$DockerUsername/$SERVICE_NAME`:$Version"
    docker tag $LOCAL_TAG $DOCKER_TAG
    Write-Host "  Tagged as: $DOCKER_TAG" -ForegroundColor Green
    Write-Host ""
    
    Write-Host "Step 4/4: Pushing to Docker Hub..." -ForegroundColor Cyan
    Write-Host "  (Make sure you're logged in: docker login)" -ForegroundColor Gray
    docker push $DOCKER_TAG
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Push successful!" -ForegroundColor Green
        Write-Host ""
        Write-Host "========================================" -ForegroundColor Green
        Write-Host "  IMAGE READY FOR DEPLOYMENT!" -ForegroundColor Green
        Write-Host "========================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "Your image:" -ForegroundColor Cyan
        Write-Host "  $DOCKER_TAG" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "  1. Go to: https://console.cloud.google.com/run?project=reset-fitness-app" -ForegroundColor White
        Write-Host "  2. Click 'CREATE SERVICE'" -ForegroundColor White
        Write-Host "  3. Select 'Deploy one revision from an existing container image'" -ForegroundColor White
        Write-Host "  4. Enter image URL: docker.io/$DOCKER_TAG" -ForegroundColor White
        Write-Host "  5. Set container port: 8080" -ForegroundColor White
        Write-Host "  6. Add environment variables:" -ForegroundColor White
        Write-Host "     - GEMINI_API_KEY" -ForegroundColor Gray
        Write-Host "     - FOOD_API_KEY" -ForegroundColor Gray
        Write-Host "  7. Click 'CREATE'" -ForegroundColor White
        Write-Host ""
    } else {
        Write-Host "  Push failed! Make sure you're logged in with 'docker login'" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Image Information:" -ForegroundColor Cyan
docker images | Select-String -Pattern $SERVICE_NAME
Write-Host ""

