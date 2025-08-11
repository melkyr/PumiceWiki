# Dockerfile

# Stage 1: Build the Go binary in a dedicated build environment
FROM golang:1.24.3-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to leverage Docker's layer caching.
# Dependencies will only be re-downloaded if these files change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application into a static binary.
# CGO_ENABLED=0 is critical for building a static binary for scratch/distroless images.
# -ldflags '-s -w' strips debug symbols, reducing the binary size.
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w' -o /app/server ./cmd/server

# Stage 2: Create the final, minimal production image
# Using a distroless image is a security best practice as it contains only the application and its runtime dependencies.
FROM gcr.io/distroless/static-debian11

# Create a dedicated directory for the app
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/server .

# Copy the migrations directory, config file, and auth model into the final image.
# The application needs these to run.
COPY migrations ./migrations
COPY config.yml ./config.yml
COPY auth_model.conf ./auth_model.conf

# Expose the port the application will run on.
# This is documentation; the actual port mapping is done in docker-compose.
EXPOSE 8080

# The command to run when the container starts.
# We use ./server because our WORKDIR is /app.
CMD ["./server"]
