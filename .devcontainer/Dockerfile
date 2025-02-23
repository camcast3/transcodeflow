# Use the official Go image as the base image.
FROM golang:1.23.6-alpine

# Install Redis CLI tools (optional, can be helpful for testing)
RUN apk update && apk add --no-cache git

# Set the working directory inside the container
WORKDIR /workspace