#!/bin/bash

# Get current user and group IDs
USER_ID=$(id -u)
GROUP_ID=$(id -g)

# Build the Docker image
echo "Building Docker image..."
docker build . -t builder

# Run the container with bind mount and proper commands
echo "Running build container..."
docker run -v $(pwd):/workspace:Z builder