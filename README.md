# Macro Path Backend

This project is a microservice-friendly Go backend for a mobile app, designed for deployment on Google Cloud Run. It uses a modular structure with each service in its own folder and a shared package for common code.

## Project Structure

- `services/` — Each microservice lives here and can be deployed independently. Use the Makefile to scaffold new services.
- `shared/` — Shared Go code for all services (e.g., Firebase initialization).
  - `firebase.go` — Placeholder for Firebase initialization logic.
  - `go.mod` — Module: `github.com/MacroPath/macro-path-backend/shared` (Go 1.24)
- `Makefile` — Run, build, test, tidy, scaffold, and deploy services from the project root.

## Usage

### Managing Services with the Makefile

All service management can be done from the project root using the Makefile. You can specify the service name with `SERVICE`, `S`, or `s`.

> **Service Name Rule:** Service names must be lower-case, dash-separated (e.g., `my-service`, `user-profile`, `workout-session`). No camelCase, underscores, or uppercase letters. This ensures compatibility with Go, Docker, and Cloud Run.

#### Run a Service in Watch Mode (Live Reload)

To automatically reload your Go app on code changes, use [air](https://github.com/cosmtrek/air):

1. Install air (if you don't have it):
   ```sh
   go install github.com/cosmtrek/air@latest
   # or
   brew install air
   # or see https://github.com/cosmtrek/air for other options
   ```
2. Run your service in watch mode (SERVICE is required):
   ```sh
   make watch SERVICE=my-service
   ```
   If you do not specify a service, you will see:
   ```
   SERVICE is not defined. Usage: make watch SERVICE=<name>, S=<name>, or s=<name>
   ```
   The service name must be lower-case, dash-separated (e.g., my-service, user-profile, workout-session).

#### Run a Service Locally

```sh
make go-run SERVICE=my-service
# or
make go-run s=my-service
```

#### Build a Service

```sh
make go-build SERVICE=my-service
```

#### Test a Service

```sh
make go-test SERVICE=my-service
```

#### Tidy Go Modules for a Service

```sh
make go-tidy SERVICE=my-service
```

#### Scaffold a New Service

```sh
make create SERVICE=my-service
```

This creates `services/my-service/` with a starter `main.go`, `go.mod` (using Go 1.24 and GitHub module paths), and `Dockerfile` ready for Cloud Run.

The generated `main.go` uses the latest Go 1.22+ HTTP patterns:

- Uses `http.NewServeMux()` and registers handlers with method patterns (e.g., `mux.HandleFunc("GET /health", ...)`).
- Passes the mux to `http.ListenAndServe`.
- Routes:
  - `GET /health` — Health check endpoint
  - `GET /` — Main endpoint for the service, handled by `rootHandler`

#### Other Useful Commands

- **Install a Go package in a service**

  ```sh
  make go-install SERVICE=my-service DEPENDENCY=github.com/some/package@latest
  # or
  make go-install s=my-service d=github.com/some/package@latest
  # or
  make go-install S=my-service D=github.com/some/package@latest
  ```

  Installs the specified Go package in the selected service. You can use `DEPENDENCY`, `D`, or `d` for the package name.

- **Clean built binaries in a service**

  ```sh
  make clean SERVICE=my-service
  ```

  Removes the built binary from the service directory.

- **Download Go modules for a service**
  ```sh
  make mod-download SERVICE=my-service
  ```
  Downloads all Go modules for the selected service.

## Cloud Run Deployment

You can deploy a service to Google Cloud Run using the Makefile. The default `PROJECT_ID` is `macro-path` and the default `REGION` is `us-central1`, but you can override them as needed.

```sh
make deploy SERVICE=my-service
# or override project/region
make deploy SERVICE=my-service PROJECT_ID=my-gcp-project REGION=europe-west1
```

**Note:**

- The deploy command will automatically authenticate Docker to Google Cloud before pushing the image (using `gcloud auth configure-docker`).
- You must have the [Google Cloud SDK (gcloud CLI)](https://cloud.google.com/sdk/docs/install) installed and authenticated (`gcloud auth login`).
- **Service names must be lower-case, dash-separated (no camelCase, underscores, or uppercase letters).**
- The handler function in the generated service is always named `rootHandler` for compatibility.
- The generated main.go uses Go 1.22+ ServeMux method patterns and passes the mux to ListenAndServe.
- **The Dockerfile build step uses `GOOS=linux` and `GOARCH=amd64` to ensure the binary is compatible with Cloud Run. This is handled automatically for all new and existing services.**

This will:

1. Build the Docker image for the service:
   ```sh
   docker build -t gcr.io/$(PROJECT_ID)/my-service:latest services/my-service
   ```
2. Authenticate Docker to Google Cloud (if not already):
   ```sh
   gcloud auth configure-docker --quiet
   ```
3. Push the image:
   ```sh
   docker push gcr.io/$(PROJECT_ID)/my-service:latest
   ```
4. Deploy to Cloud Run:
   ```sh
   gcloud run deploy my-service \
     --image gcr.io/$(PROJECT_ID)/my-service:latest \
     --platform managed \
     --region $(REGION) \
     --allow-unauthenticated
   ```

## Testing a Private (Authenticated) Cloud Run Service

By default, deployed services require authentication. To test your service, you need to fetch an identity token and include it in your requests.

### Using curl

```sh
SERVICE_URL=https://<your-service-url>
curl -H "Authorization: Bearer $(gcloud auth print-identity-token)" $SERVICE_URL
```

### Using Thunder Client, Postman, or Other API Clients

1. Run:
   ```sh
   gcloud auth print-identity-token
   ```
2. Copy the output token.
3. In your API client, set the `Authorization` header to:
   ```
   Bearer <paste-your-token-here>
   ```
4. Make your request to the service URL.

---

You can add more services under `services/` using the Makefile. All services and shared code use Go 1.24 and follow Go module best practices with GitHub-based import paths.
