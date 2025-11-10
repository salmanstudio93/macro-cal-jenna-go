# Start from the latest golang base image
FROM golang:1.24 as builder

WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o service-two

# Use a minimal base image for the final container
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/service-two ./
EXPOSE 8080
CMD ["/app/service-two"]

