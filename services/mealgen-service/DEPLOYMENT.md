# Deployment Guide for mealgen-service

This guide explains how to deploy the mealgen-service to Google Cloud Run.

## Prerequisites

1. **Google Cloud SDK** installed and authenticated
2. **Docker** installed
3. **API Keys** for Gemini and Food services

## Quick Deployment

### 1. Set Environment Variables

```bash
# Set your API keys
export GEMINI_API_KEY="your_gemini_api_key_here"
export FOOD_API_KEY="your_food_api_key_here"

# Verify they are set
echo "GEMINI_API_KEY: $GEMINI_API_KEY"
echo "FOOD_API_KEY: $FOOD_API_KEY"
```

### 2. Deploy to Cloud Run

```bash
# Using Makefile (recommended)
make deploy SERVICE=mealgen-service

# Or using the deployment script
./deploy-cloudrun.sh deploy
```

## Detailed Steps

### Step 1: Environment Setup

The service requires two environment variables:

- `GEMINI_API_KEY`: Your Google Gemini API key
- `FOOD_API_KEY`: Your Food API key

**Important**: These must be set in your shell environment before running the deployment command.

### Step 2: Build and Deploy

The deployment process will:

1. **Build** the Docker image
2. **Push** to Google Container Registry
3. **Deploy** to Cloud Run with environment variables

### Step 3: Verify Deployment

After deployment, you can:

1. **Check the service URL** (displayed after deployment)
2. **Test the health endpoint**: `curl <service-url>/health`
3. **View logs**: `gcloud run services logs read mealgen-service --region=us-central1`

## Troubleshooting

### Common Issues

#### 1. "Environment variable is required" Error

**Problem**: Service fails to start with missing API keys.

**Solution**:

```bash
# Make sure environment variables are set
export GEMINI_API_KEY="your_key"
export FOOD_API_KEY="your_key"

# Verify they are exported
env | grep API_KEY
```

#### 2. "Container failed to start" Error

**Problem**: Container fails to start and listen on port.

**Possible Causes**:

- Missing environment variables
- Application crash during startup
- Port binding issues

**Solution**:

1. Check Cloud Run logs: `gcloud run services logs read mealgen-service --region=us-central1`
2. Verify environment variables are set
3. Test locally first: `docker run --env-file .env -p 8080:8080 mealgen-service:latest`

#### 3. "Revision is not ready" Error

**Problem**: Cloud Run revision fails to become ready.

**Solution**:

1. Check the logs URL provided in the error message
2. Verify all environment variables are set
3. Ensure the application starts successfully locally

### Debugging Steps

1. **Test Locally**:

   ```bash
   # Build the image
   docker build -t mealgen-service:test .

   # Run with environment variables
   docker run -e GEMINI_API_KEY="$GEMINI_API_KEY" -e FOOD_API_KEY="$FOOD_API_KEY" -p 8080:8080 mealgen-service:test
   ```

2. **Check Logs**:

   ```bash
   # View recent logs
   gcloud run services logs read mealgen-service --region=us-central1 --limit=50

   # Follow logs in real-time
   gcloud run services logs tail mealgen-service --region=us-central1
   ```

3. **Verify Environment Variables**:
   ```bash
   # Check if variables are set in your shell
   echo $GEMINI_API_KEY
   echo $FOOD_API_KEY
   ```

## Environment Variables Reference

| Variable         | Description           | Required | Default | Notes                           |
| ---------------- | --------------------- | -------- | ------- | ------------------------------- |
| `GEMINI_API_KEY` | Google Gemini API key | Yes      | -       | Get from Google AI Studio       |
| `FOOD_API_KEY`   | Food API key          | Yes      | -       | Get from your food API provider |
| `PORT`           | Port to listen on     | No       | 8080    | Set automatically by Cloud Run  |
| `LOG_LEVEL`      | Logging level         | No       | info    | -                               |

## Security Notes

1. **Never commit API keys** to version control
2. **Use environment variables** for sensitive data
3. **Consider using Google Secret Manager** for production
4. **Rotate API keys** regularly

## Production Considerations

1. **Use Google Secret Manager** for API keys in production
2. **Set up monitoring** and alerting
3. **Configure proper IAM** permissions
4. **Set up CI/CD** pipeline for automated deployments
5. **Use different environments** (dev, staging, prod)

## Example Production Deployment

```bash
# Set up secrets in Google Secret Manager
gcloud secrets create gemini-api-key --data-file=- <<< "your_gemini_key"
gcloud secrets create food-api-key --data-file=- <<< "your_food_key"

# Deploy with secrets
gcloud run deploy mealgen-service \
  --image gcr.io/macro-path/mealgen-service:latest \
  --platform managed \
  --region us-central1 \
  --set-secrets="GEMINI_API_KEY=gemini-api-key:latest,FOOD_API_KEY=food-api-key:latest" \
  --no-allow-unauthenticated
```
