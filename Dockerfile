# Use a Debian base image
FROM debian:bullseye

ARG TARGETPLATFORM

# Set the working directory
WORKDIR /workspace

# Install necessary packages
RUN apt-get update && apt-get install -y \
    curl \
    git \
    make \
    wget \
    unzip \
    gnupg \
    xvfb \
    bash \
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

# Install chromedriver
RUN apt-get install -y chromium && \
    CHROMIUM_VERSION=$(chromium --version | grep -oP '\d+\.\d+\.\d+') && \
    CHROMEDRIVER_VERSION=$(wget -qO- "https://chromedriver.storage.googleapis.com/LATEST_RELEASE_$CHROMIUM_VERSION") && \
    wget -N "http://chromedriver.storage.googleapis.com/$CHROMEDRIVER_VERSION/chromedriver_linux64.zip" && \
    unzip chromedriver_linux64.zip && \
    chmod +x chromedriver && \
    mv chromedriver /usr/local/bin/ && \
    rm chromedriver_linux64.zip

# Clean up
RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# The container starts with a shell by default; adjust as necessary for your workflow
CMD ["/bin/bash"]
