FROM debian:bullseye


WORKDIR /workspace

RUN apt-get update && apt-get install -y \
    curl \
    git \
    make \
    wget \
    unzip \
    gnupg \
    xvfb \
    bash \
    pulseaudio \
    ffmpeg \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://golang.org/dl/go1.22.5.linux-amd64.tar.gz \
    && tar -xvf go1.22.5.linux-amd64.tar.gz \
    && mv go /usr/local \
    && rm go1.22.5.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/main cmd/*.go

RUN echo "deb http://deb.debian.org/debian bullseye main contrib non-free" > /etc/apt/sources.list && \
    echo "deb http://deb.debian.org/debian-security/ bullseye-security main contrib non-free" >> /etc/apt/sources.list && \
    echo "deb http://deb.debian.org/debian bullseye-updates main contrib non-free" >> /etc/apt/sources.list && \
    apt-get update && apt-get install -y chromium chromium-driver

RUN CHROMIUM_PATH=$(which chromium || which chromium-browser) && \
    echo "Found chromium at $CHROMIUM_PATH" && \
    if [ -z "$CHROMIUM_PATH" ]; then echo "Chromium not found in PATH" && exit 1; fi && \
    PATH=$PATH:$(dirname $CHROMIUM_PATH)

RUN FFMPEG_PATH=$(which ffmpeg) && \
    echo "Found ffmpeg at $FFMPEG_PATH" && \
    if [ -z "$FFMPEG_PATH" ]; then echo "FFmpeg not found in PATH" && exit 1; fi && \
    PATH=$PATH:$(dirname $FFMPEG_PATH)

ENV PATH=$PATH

# Clean up
RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN chromium --version && \
    chromedriver --version && \
    ffmpeg -version

# Add pulseaudio as a root user.
RUN adduser root pulse-access

COPY pulseaudio.sh .

ENTRYPOINT ["./pulseaudio.sh"]