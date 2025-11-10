# Macro Path Backend Makefile
#
# This Makefile provides commands to manage Go microservices in the /services directory.
# You can run, build, test, tidy, scaffold new services, and deploy to Cloud Run from the project root.
#
# Usage examples:
#   make go-run SERVICE=my-service
#   make go-build S=my-service
#   make go-run s=my-service
#   make create SERVICE=my-service
#   make create s=my-service
#   make deploy SERVICE=my-service
#
# Variables:
#   SERVICE, S, or s: Name of the service (directory under /services). Must be lower-case, dash-separated (e.g., my-service, user-profile, workout-session).
#   PROJECT_ID: Google Cloud project ID (default: macro-path)
#   REGION: Google Cloud region (default: us-central1)

PROJECT_ID ?= macro-path
REGION ?= us-central1

.PHONY: go-run go-build go-test go-tidy create deploy watch go-install clean mod-download

# Robustly assign SERVICE from SERVICE, S, or s (in that order)
SERVICE := $(shell [ -n "$(SERVICE)" ] && echo $(SERVICE) || ([ -n "$(S)" ] && echo $(S) || echo $(s)))

# Robustly assign DEP from DEPENDENCY, D, or d (in that order)
DEP := $(shell [ -n "$(DEPENDENCY)" ] && echo $(DEPENDENCY) || ([ -n "$(D)" ] && echo $(D) || echo $(d)))

# Error if SERVICE is still not set
ifeq ($(strip $(SERVICE)),)
$(error SERVICE is not defined. Usage: make go-<target> SERVICE=<name>, S=<name>, or s=<name>)
endif

# Path to the selected service
SERVICE_PATH = services/$(SERVICE)

# Validate service name for lower-case, dash-separated (Cloud Run and Docker compatible)
VALID_DASH_SERVICE := $(shell echo $(SERVICE) | grep -E '^[a-z0-9]([-a-z0-9]*[a-z0-9])?$$' || echo INVALID)

# Run the main.go of the selected service
# Example: make go-run SERVICE=my-service
# Runs: go run main.go in services/<service>
go-run:
	@echo "🚀 Running $(SERVICE) service..."
	cd $(SERVICE_PATH) && go run main.go

# Build the selected service
# Example: make go-build SERVICE=my-service
# Runs: go build -o <service> in services/<service>
go-build:
	@echo "🔨 Building $(SERVICE) service..."
	cd $(SERVICE_PATH) && go build -o $(SERVICE)

# Run tests for the selected service
# Example: make go-test SERVICE=my-service
go-test:
	@echo "🧪 Running tests for $(SERVICE) service..."
	cd $(SERVICE_PATH) && go test ./...

# Run go mod tidy for the selected service
# Example: make go-tidy SERVICE=my-service
go-tidy:
	@echo "🧹 Tidying Go modules for $(SERVICE) service..."
	cd $(SERVICE_PATH) && go mod tidy

# Run the main.go of the selected service in watch mode using air
# Example: make watch SERVICE=my-service
watch:
	@if [ -z "$(SERVICE)" ]; then \
		echo "SERVICE is not defined. Usage: make watch SERVICE=<name>, S=<name>, or s=<name>"; \
		exit 1; \
	fi
	@if [ "$(VALID_DASH_SERVICE)" = "INVALID" ]; then \
		echo "ERROR: SERVICE name '$(SERVICE)' is invalid. Use lower-case, dash-separated (e.g., my-service, user-profile, workout-session)."; \
		exit 1; \
	fi
	@echo "👀 Watching for file changes using Air..."
	cd $(SERVICE_PATH) && air && rm -rf tmp

# Scaffold a new service with main.go, go.mod, and Dockerfile
# Example: make create SERVICE=my-service
# Creates: services/<service>/ with starter files
create:
	@if [ -z "$(SERVICE)" ]; then \
		echo "SERVICE is not defined. Usage: make create SERVICE=<name>, S=<name>, or s=<name>"; \
		exit 1; \
	fi
	@if [ "$(VALID_DASH_SERVICE)" = "INVALID" ]; then \
		echo "ERROR: SERVICE name '$(SERVICE)' is invalid. Use lower-case, dash-separated (e.g., my-service, user-profile, workout-session)."; \
		exit 1; \
	fi
	@if [ -d services/$(SERVICE) ]; then \
		echo "Service '$(SERVICE)' already exists."; \
		exit 1; \
	fi
	@echo "🏗️  Creating new service: $(SERVICE)"
	@mkdir -p services/$(SERVICE)
	@echo 'package main\n\nimport (\n\t"fmt"\n\t"log"\n\t"net/http"\n\t"os"\n)\n\nfunc healthHandler(w http.ResponseWriter, r *http.Request) {\n\tw.WriteHeader(http.StatusOK)\n\tfmt.Fprintln(w, "OK")\n}\n\nfunc rootHandler(w http.ResponseWriter, r *http.Request) {\n\tw.WriteHeader(http.StatusOK)\n\tfmt.Fprintln(w, "$(SERVICE) endpoint")\n}\n\nfunc main() {\n\tport := os.Getenv("PORT")\n\tif port == "" {\n\t\tport = "8080"\n\t}\n\tmux := http.NewServeMux()\n\tmux.HandleFunc("GET /health", healthHandler)\n\tmux.HandleFunc("GET /", rootHandler)\n\tlog.Printf("Starting $(SERVICE) service on port %s", port)\n\tlog.Fatal(http.ListenAndServe(":"+port, mux))\n}' > services/$(SERVICE)/main.go
	@echo 'module github.com/MacroPath/macro-path-backend/services/$(SERVICE)\n\ngo 1.24\n\nrequire github.com/MacroPath/macro-path-backend/shared v0.0.0\n\nreplace github.com/MacroPath/macro-path-backend/shared => ../../shared\n' > services/$(SERVICE)/go.mod
	@echo '# Start from the latest golang base image\nFROM golang:1.24 as builder\n\nWORKDIR /app\nCOPY . .\nRUN GOOS=linux GOARCH=amd64 go build -o $(SERVICE)\n\n# Use a minimal base image for the final container\nFROM gcr.io/distroless/base-debian12\nWORKDIR /app\nCOPY --from=builder /app/$(SERVICE) ./\nEXPOSE 8080\n# Use PORT environment variable for Cloud Run compatibility\nENV PORT=8080\nCMD ["/app/$(SERVICE)"]\n' > services/$(SERVICE)/Dockerfile
	@echo "✅ Service $(SERVICE) created successfully!"

# Deploy a specific service to Google Cloud Run
# Usage: make deploy SERVICE=my-service [PROJECT_ID=your-gcp-project] [REGION=your-region]
# This target ensures Docker is authenticated with Google Cloud before pushing.
# Uses SERVICE for Docker image tag, Cloud Run, and Go code (all lower-case, dash-separated).
deploy:
	@if [ "$(VALID_DASH_SERVICE)" = "INVALID" ]; then \
		echo "ERROR: SERVICE name '$(SERVICE)' is invalid. Use lower-case, dash-separated (e.g., my-service, user-profile, workout-session)."; \
		exit 1; \
	fi
	@echo "🚀 Deploying $(SERVICE) to Google Cloud Run..."
	@echo "🔐 Authenticating Docker to Google Cloud..."
	@gcloud auth configure-docker --quiet
	@echo "📁 Setting gcloud project to macro-path..."
	@gcloud config set project macro-path
	@echo "🔨 Building Docker image..."
	docker build -t gcr.io/$(PROJECT_ID)/$(SERVICE):latest services/$(SERVICE)
	@echo "📤 Pushing Docker image..."
	docker push gcr.io/$(PROJECT_ID)/$(SERVICE):latest
	@echo "☁️  Deploying to Cloud Run..."
	gcloud run deploy $(SERVICE) \
		--image gcr.io/$(PROJECT_ID)/$(SERVICE):latest \
		--platform managed \
		--region $(REGION) \
		--port 8080 \
		--memory 1Gi \
		--cpu 1 \
		--min-instances 0 \
		--max-instances 10 \
		--timeout 300 \
		--concurrency 80 \
		--allow-unauthenticated
	@echo "✅ $(SERVICE) deployed successfully!"

# Install a Go package in the selected service
# Example: make go-install SERVICE=my-service DEPENDENCY=github.com/some/package@latest
#          make go-install s=my-service d=github.com/some/package@latest
#          make go-install S=my-service D=github.com/some/package@latest
go-install:
	@if [ -z "$(SERVICE)" ]; then \
		echo "SERVICE is not defined. Usage: make go-install SERVICE=<name> DEPENDENCY=<package>"; \
		exit 1; \
	fi
	@if [ -z "$(DEP)" ]; then \
		echo "DEPENDENCY is not defined. Usage: make go-install SERVICE=<name> DEPENDENCY=<package>"; \
		exit 1; \
	fi
	@echo "📦 Installing $(DEP) in $(SERVICE) service..."
	cd $(SERVICE_PATH) && go get $(DEP)
	@echo "✅ Package installed successfully!"

# Clean built binaries in the selected service
# Example: make clean SERVICE=my-service
clean:
	@if [ -z "$(SERVICE)" ]; then \
		echo "SERVICE is not defined. Usage: make clean SERVICE=<name>"; \
		exit 1; \
	fi
	@echo "🧹 Cleaning built binaries for $(SERVICE) service..."
	cd $(SERVICE_PATH) && rm -f $(SERVICE)
	@echo "✅ Cleaned successfully!"

# Download Go modules for the selected service
# Example: make mod-download SERVICE=my-service
mod-download:
	@if [ -z "$(SERVICE)" ]; then \
		echo "SERVICE is not defined. Usage: make mod-download SERVICE=<name>"; \
		exit 1; \
	fi
	@echo "📥 Downloading Go modules for $(SERVICE) service..."
	cd $(SERVICE_PATH) && go mod download
	@echo "✅ Modules downloaded successfully!"
