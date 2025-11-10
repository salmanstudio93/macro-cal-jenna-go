# Start from the latest golang base image
FROM golang:1.24 as builder

WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o mealgen-service

# Use a minimal base image for the final container
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/mealgen-service ./
EXPOSE 8080
# Use PORT environment variable for Cloud Run compatibility
ENV PORT=8080
CMD ["/app/mealgen-service"]

