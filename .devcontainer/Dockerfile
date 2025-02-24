# Use the official Go image as the base image.
FROM golang:1.23.6-bookworm

# Install Redis CLI tools (optional, can be helpful for testing)
RUN apt update && apt -y install git redis-tools

# Set the working directory inside the container
WORKDIR /workspace