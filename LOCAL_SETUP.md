# Local Development Setup Guide

## Prerequisites

1. **Go 1.24** installed ([Download Go](https://golang.org/dl/))
2. **API Keys**:
   - Gemini API Key (from Google AI Studio)
   - Food API Key (from FatSecret or similar nutrition API)

## Step 1: Set Up Environment Variables

1. Navigate to the service directory:
```powershell
cd "d:\Studio93\macro calculator web\macro-path-backend-mealgen-service\services\mealgen-service"
```

2. Create a `.env` file in the `mealgen-service` directory:
```powershell
Copy-Item .env.example .env
```

3. Edit the `.env` file and add your API keys:
```env
GEMINI_API_KEY=your_actual_gemini_api_key
FOOD_API_KEY=your_actual_food_api_key
PORT=8080
```

## Step 2: Install Dependencies

```powershell
go mod tidy
go mod download
```

## Step 3: Run the Service Locally

### Option A: Direct Run
```powershell
go run main.go
```

### Option B: Build and Run
```powershell
go build -o mealgen-service.exe
./mealgen-service.exe
```

### Option C: Using Make (from project root)
```powershell
cd ../..
make go-run SERVICE=mealgen-service
```

### Option D: Watch Mode (Auto-reload on changes)
First, install Air:
```powershell
go install github.com/cosmtrek/air@latest
```

Then run:
```powershell
cd ../..
make watch SERVICE=mealgen-service
```

## Step 4: Verify the Service is Running

Open a browser or use curl:
```powershell
# Health check
curl http://localhost:8080/health

# Root endpoint
curl http://localhost:8080/
```

You should see:
- Health check: `OK`
- Root: `mealgen-service endpoint`

## Step 5: Update Your Next.js Project

In your Next.js project, update the API URL from the Cloud Run URL to your local URL:

**Before:**
```typescript
const eventSource = new EventSource(
  `https://apidevchunk-p6eeewaenq-uc.a.run.app/program/generate-program?payload=${encoded}`
);
```

**After (for local development):**
```typescript
const eventSource = new EventSource(
  `http://localhost:8080/program/generate-program?payload=${encoded}`
);
```

## Available Endpoints

1. **GET /health** - Health check endpoint
2. **GET /** - Service info
3. **POST /** - Generate meal plan (main endpoint)
4. **POST /regenerate** - Regenerate a specific meal

## Testing the Meal Generation Endpoint

Create a test JSON file or use curl:

```powershell
curl -X POST http://localhost:8080/ `
  -H "Content-Type: application/json" `
  -d '{
    "dates": ["2025-01-01", "2025-01-02"],
    "diet_type": "Balanced",
    "meal_style": "American",
    "number_of_meals": 3,
    "macro_target": {
      "calories": 2000,
      "proteins": 150,
      "carbs": 200,
      "fats": 65
    },
    "foods_to_avoid": [],
    "foods_to_like": ["chicken", "rice", "broccoli"]
  }'
```

## Troubleshooting

### Port Already in Use
If port 8080 is already in use, change it in your `.env` file:
```env
PORT=8081
```

### Missing Dependencies
```powershell
go mod tidy
go mod download
```

### API Key Errors
Make sure your `.env` file is in the correct location:
```
macro-path-backend-mealgen-service/
└── services/
    └── mealgen-service/
        ├── .env          ← Should be here
        ├── main.go
        └── ...
```

### CORS Issues (if calling from Next.js)
If you encounter CORS issues when calling from your Next.js frontend, you may need to add CORS headers to the Go backend. See the CORS configuration section below.

## Next Steps

- Keep the Go service running in one terminal
- Run your Next.js app in another terminal
- The Next.js app will now use your local Go backend instead of Cloud Run

## Stopping the Service

Press `Ctrl+C` in the terminal where the service is running.

