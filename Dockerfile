# Use a Debian base image
FROM debian:bullseye

ARG TARGETPLATFORM

# Set the working directory
WORKDIR /workspace

# Install necessary packages including FFmpeg
RUN apt-get update && apt-get install -y \
    curl \
    git \
    make \
    wget \
    unzip \
    gnupg \
    xvfb \
    bash \
    ffmpeg \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Golang 1.22.5
RUN wget https://golang.org/dl/go1.22.5.linux-amd64.tar.gz \
    && tar -xvf go1.22.5.linux-amd64.tar.gz \
    && mv go /usr/local \
    && rm go1.22.5.linux-amd64.tar.gz

# Add Go to PATH
ENV PATH="/usr/local/go/bin:${PATH}"

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app using Go or Make, depending on your setup
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/main cmd/*.go

# Install Chromium and ChromeDriver
RUN echo "deb http://deb.debian.org/debian bullseye main contrib non-free" > /etc/apt/sources.list && \
    echo "deb http://deb.debian.org/debian-security/ bullseye-security main contrib non-free" >> /etc/apt/sources.list && \
    echo "deb http://deb.debian.org/debian bullseye-updates main contrib non-free" >> /etc/apt/sources.list && \
    apt-get update && apt-get install -y chromium chromium-driver

# Find Chromium installation path and update PATH environment
RUN CHROMIUM_PATH=$(which chromium || which chromium-browser) && \
    echo "Found chromium at $CHROMIUM_PATH" && \
    if [ -z "$CHROMIUM_PATH" ]; then echo "Chromium not found in PATH" && exit 1; fi && \
    PATH=$PATH:$(dirname $CHROMIUM_PATH)

# Verify FFmpeg installation and add to PATH if necessary
RUN FFMPEG_PATH=$(which ffmpeg) && \
    echo "Found ffmpeg at $FFMPEG_PATH" && \
    if [ -z "$FFMPEG_PATH" ]; then echo "FFmpeg not found in PATH" && exit 1; fi && \
    PATH=$PATH:$(dirname $FFMPEG_PATH)

# Set the updated PATH permanently
ENV PATH=$PATH

# Clean up
RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Verify installations
RUN chromium --version && \
    chromedriver --version && \
    ffmpeg -version

# The container starts with a shell by default; adjust as necessary for your workflow
CMD ["/bin/bash"]