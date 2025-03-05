# Use the official Golang image as the base image
FROM golang:1.24-alpine AS builder

# Set the working directory
WORKDIR /app

# Install dependencies including git and curl
RUN apk add --no-cache git curl

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod tidy
RUN go mod download

# Install Goose for database migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Install Swag for Swagger documentation
RUN go get github.com/swaggo/swag/cmd/swag
RUN go install github.com/swaggo/swag/cmd/swag
RUN go get github.com/swaggo/gin-swagger
RUN go install github.com/swaggo/gin-swagger


# Copy the rest of the application
COPY . .

# Generate Swagger docs during the build stage
RUN swag init --parseDependency --parseInternal

# Build the Go application as a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o my-backend-app .

# Final stage with minimal image
FROM alpine:3.21.3

# Set working directory
WORKDIR /app

# Install Go in the final image (needed for Goose and other build tools)
RUN apk --no-cache add go git ca-certificates

# Copy the built binary from the builder stage
COPY --from=builder /app/my-backend-app .

# Copy migration files
COPY --from=builder /app/migrations ./migrations

# Copy the Swagger generated docs
COPY --from=builder /app/docs ./docs

# Copy the goose binary from the builder stage to the final image
COPY --from=builder /go/bin/goose /usr/local/bin/goose

# Copy config files
COPY config /app/config

# Ensure execution permissions
RUN chmod +x /app/my-backend-app

# Expose the application's port
EXPOSE 8080

# Run Goose migrations before starting the app
CMD ["sh", "-c", "goose -dir migrations postgres $DATABASE_URL up && ./my-backend-app"] 