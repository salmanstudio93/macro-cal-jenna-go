# 🚀 Deploy mealgen-service to Google Cloud Run

## ✅ Status: Docker Image Built & Tested

Your `mealgen-service` Docker image is built and tested locally on port 8090.

**Image:** `mealgen-service:local` (49.9MB)
**Status:** Running and healthy ✅

---

## 📦 Option 1: Deploy via Docker Hub (Easiest - No gcloud needed!)

### Step 1: Login to Docker Hub

```powershell
# Login (you'll be prompted for username/password)
docker login
```

If you don't have a Docker Hub account, create one free at: https://hub.docker.com/signup

### Step 2: Tag and Push Your Image

```powershell
# Replace 'yourusername' with your Docker Hub username
$DOCKER_USERNAME = "yourusername"  # CHANGE THIS!

# Tag the image
docker tag mealgen-service:local $DOCKER_USERNAME/mealgen-service:latest

# Push to Docker Hub
docker push $DOCKER_USERNAME/mealgen-service:latest
```

### Step 3: Deploy via Google Cloud Console

1. **Open Cloud Run Console:**
   ```
   https://console.cloud.google.com/run?project=reset-fitness-app
   ```

2. **Click "CREATE SERVICE"**

3. **Select deployment source:**
   - Choose "Deploy one revision from an existing container image"
   - Click "SELECT"

4. **Enter your image URL:**
   ```
   docker.io/yourusername/mealgen-service:latest
   ```
   (Replace `yourusername` with your actual Docker Hub username)

5. **Service Configuration:**
   - **Service name:** `mealgen-service`
   - **Region:** `us-central1` (or your preferred region)

6. **Container Settings** (click "CONTAINER(S), VOLUMES, NETWORKING, SECURITY"):
   
   **Container tab:**
   - **Container port:** `8080`
   - **Memory:** `1 GiB`
   - **CPU:** `1`

   **Variables & Secrets tab:**
   Click "ADD VARIABLE" and add these:
   - `GEMINI_API_KEY` = `AIzaSyD7C2E3682Avm36w7OynNP_4j9DaX8tXTw`
   - `FOOD_API_KEY` = `83b88a97d33c889f3fa582d1d190d53b2655a72da9353465f70fbaaa18d498a0`

7. **General settings:**
   - **Request timeout:** `300` seconds
   - **Maximum concurrent requests:** `80`

8. **Autoscaling:**
   - **Minimum instances:** `0`
   - **Maximum instances:** `10`

9. **Authentication:**
   - Select "**Allow unauthenticated invocations**"

10. **Click "CREATE"**

11. **Wait** ~2-3 minutes for deployment

12. **Copy your service URL** (will look like: `https://mealgen-service-xxxxx-uc.a.run.app`)

---
https://mealgen-service-544921973826.europe-west1.run.app
## 📦 Option 2: Deploy via Google Container Registry (if you have gcloud)

### Step 1: Tag and Push to GCR

```powershell
# Set variables
$PROJECT_ID = "reset-fitness-app"
$SERVICE_NAME = "mealgen-service"

# Configure Docker for GCR
gcloud auth configure-docker --quiet

# Tag for GCR
docker tag mealgen-service:local gcr.io/$PROJECT_ID/$SERVICE_NAME:latest

# Push to GCR
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:latest
```

### Step 2: Deploy to Cloud Run

```powershell
gcloud run deploy mealgen-service `
    --image gcr.io/reset-fitness-app/mealgen-service:latest `
    --platform managed `
    --region us-central1 `
    --project reset-fitness-app `
    --port 8080 `
    --memory 1Gi `
    --cpu 1 `
    --min-instances 0 `
    --max-instances 10 `
    --timeout 300 `
    --concurrency 80 `
    --allow-unauthenticated `
    --set-env-vars GEMINI_API_KEY="AIzaSyD7C2E3682Avm36w7OynNP_4j9DaX8tXTw",FOOD_API_KEY="83b88a97d33c889f3fa582d1d190d53b2655a72da9353465f70fbaaa18d498a0"
```

---

## 🧪 Testing Your Deployment

Once deployed, test your service:

```powershell
# Replace with your actual service URL
$SERVICE_URL = "https://mealgen-service-xxxxx-uc.a.run.app"

# Test health check
Invoke-RestMethod -Uri "$SERVICE_URL/health"

# Test meal generation
$testBody = @{
    name = "Test User"
    age = 30
    gender = "male"
    weight = 75
    height = 175
    goal = "muscle gain"
    activityLevel = "moderate"
    dietType = "balanced"
    mealsPerDay = "3"
} | ConvertTo-Json

Invoke-RestMethod -Uri $SERVICE_URL -Method Post -Body $testBody -ContentType "application/json" -TimeoutSec 120
```

---

## 🔄 Updating Your Deployment

When you make code changes:

1. **Rebuild the image:**
   ```powershell
   docker build -t mealgen-service:local ./services/mealgen-service
   ```

2. **Push the new version:**
   ```powershell
   # If using Docker Hub:
   docker tag mealgen-service:local yourusername/mealgen-service:latest
   docker push yourusername/mealgen-service:latest
   
   # If using GCR:
   docker tag mealgen-service:local gcr.io/reset-fitness-app/mealgen-service:latest
   docker push gcr.io/reset-fitness-app/mealgen-service:latest
   ```

3. **Redeploy:**
   - Via Console: Go to Cloud Run → Select service → "EDIT & DEPLOY NEW REVISION"
   - Via gcloud: Run the same `gcloud run deploy` command again

---

## 🧹 Clean Up Local Container

```powershell
# Stop and remove test container
docker stop mealgen-test
docker rm mealgen-test

# Optional: Remove local image to save space
docker rmi mealgen-service:local
```

---

## 📋 Available Endpoints

Once deployed, your service will have these endpoints:

- `GET  /health` - Health check
- `GET  /` - Root endpoint
- `POST /` - Generate meal plan
- `POST /regenerate` - Regenerate specific meal
- `POST /program/generate-program` - Generate program with SSE streaming

---

## 💰 Cost Estimate

**Google Cloud Run Pricing:**
- First 2 million requests/month: **FREE**
- CPU/Memory only charged during request processing
- With min instances = 0: **$0 when idle**
- Estimated cost for moderate use: **$0-5/month**

**Docker Hub:**
- Free tier: Unlimited public repositories
- Rate limits: 100 pulls per 6 hours (sufficient for updates)

---

## 🔒 Security Best Practice (Optional)

For production, use Google Secret Manager:

```powershell
# Create secrets
echo "AIzaSyD7C2E3682Avm36w7OynNP_4j9DaX8tXTw" | gcloud secrets create gemini-api-key --data-file=- --project=reset-fitness-app
echo "83b88a97d33c889f3fa582d1d190d53b2655a72da9353465f70fbaaa18d498a0" | gcloud secrets create food-api-key --data-file=- --project=reset-fitness-app

# Grant access
gcloud secrets add-iam-policy-binding gemini-api-key `
    --member="serviceAccount:544921973826-compute@developer.gserviceaccount.com" `
    --role="roles/secretmanager.secretAccessor" `
    --project=reset-fitness-app

gcloud secrets add-iam-policy-binding food-api-key `
    --member="serviceAccount:544921973826-compute@developer.gserviceaccount.com" `
    --role="roles/secretmanager.secretAccessor" `
    --project=reset-fitness-app

# Deploy with secrets
gcloud run deploy mealgen-service `
    --image gcr.io/reset-fitness-app/mealgen-service:latest `
    --set-secrets="GEMINI_API_KEY=gemini-api-key:latest,FOOD_API_KEY=food-api-key:latest" `
    --project=reset-fitness-app `
    --region=us-central1 `
    --allow-unauthenticated
```

---

**Next Step:** Choose Option 1 (Docker Hub) or Option 2 (GCR) and follow the steps above!

