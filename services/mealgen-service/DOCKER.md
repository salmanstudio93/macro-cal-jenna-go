# Docker Setup for mealgen-service

This document explains how to run the mealgen-service using Docker with environment variables from a .env file.

## Prerequisites

- Docker and Docker Compose installed
- API keys for Gemini and Food services

## Quick Start

1. **Set up environment variables:**

   ```bash
   # Copy the example environment file
   cp .env.example .env

   # Edit .env with your actual API keys
   nano .env
   ```

2. **Run with Docker Compose (Recommended):**

   ```bash
   # Build and start the service
   ./docker-helper.sh run

   # Or use docker-compose directly
   docker-compose up --build
   ```

3. **Run with Docker directly:**

   ```bash
   # Build the image
   ./docker-helper.sh build

   # Run the container
   ./docker-helper.sh docker
   ```

## Environment Variables

The service requires the following environment variables:

| Variable         | Description                | Required | Default | Notes                          |
| ---------------- | -------------------------- | -------- | ------- | ------------------------------ |
| `GEMINI_API_KEY` | API key for Gemini service | Yes      | -       | -                              |
| `FOOD_API_KEY`   | API key for Food service   | Yes      | -       | -                              |
| `PORT`           | Port to run the service on | No       | 8080    | Set automatically by Cloud Run |
| `LOG_LEVEL`      | Logging level              | No       | info    | -                              |

## Docker Commands

### Using the helper script:

```bash
./docker-helper.sh build     # Build Docker image
./docker-helper.sh run       # Run with docker-compose
./docker-helper.sh docker    # Run with Docker directly
./docker-helper.sh stop      # Stop containers
./docker-helper.sh logs      # Show logs
./docker-helper.sh cleanup   # Clean up Docker resources
```

### Using Docker directly:

```bash
# Build the image
docker build -t mealgen-service:latest .

# Run with environment file
docker run --env-file .env -p 8080:8080 mealgen-service:latest

# Run with environment variables
docker run -e GEMINI_API_KEY=your_key -e FOOD_API_KEY=your_key -p 8080:8080 mealgen-service:latest
```

### Using Docker Compose:

```bash
# Start services
docker-compose up --build

# Start in background
docker-compose up -d --build

# Stop services
docker-compose down

# View logs
docker-compose logs -f
```

## Health Check

The service includes a health check endpoint at `/health`. You can test it:

```bash
curl http://localhost:8080/health
```

## Google Cloud Run Deployment

For production deployment to Google Cloud Run:

1. **Set up environment variables:**

   ```bash
   # Use the setup script
   ./setup-env.sh setup

   # Or set manually
   export GEMINI_API_KEY=your_gemini_api_key
   export FOOD_API_KEY=your_food_api_key
   ```

2. **Deploy to Cloud Run:**

   ```bash
   # Use the Cloud Run deployment script
   ./deploy-cloudrun.sh deploy

   # Or use the Makefile
   make deploy SERVICE=mealgen-service
   ```

3. **Environment Variables for Cloud Run:**
   - Cloud Run passes environment variables directly to the container
   - No .env file needed in the container
   - Variables are set via `gcloud run deploy --set-env-vars`
   - **Note**: `PORT` is automatically set by Cloud Run - don't override it

## Production Considerations

1. **Security**: Never commit .env files with real API keys to version control
2. **Secrets Management**: For production, consider using Google Secret Manager
3. **Resource Limits**: Set appropriate CPU and memory limits in Cloud Run
4. **Logging**: Configure log rotation and centralized logging
5. **Monitoring**: Set up health checks and monitoring
6. **Cloud Run**: Use the provided deployment scripts for Cloud Run

## Troubleshooting

### Common Issues:

1. **Missing .env file**: The helper script will create one from .env.example
2. **Port conflicts**: Change the port in docker-compose.yml or use -p flag
3. **API key errors**: Verify your API keys are correct in the .env file
4. **Build failures**: Check Dockerfile and ensure all dependencies are available

### Debug Commands:

```bash
# Check container logs
docker-compose logs mealgen-service

# Inspect running container
docker exec -it <container_id> sh

# Check environment variables
docker exec <container_id> env
```

## File Structure

```
mealgen-service/
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.yml      # Docker Compose configuration
├── .dockerignore           # Files to ignore during build
├── docker-helper.sh        # Helper script for Docker operations
├── .env.example            # Example environment file
├── .env                    # Your environment file (create this)
└── DOCKER.md              # This documentation
```
