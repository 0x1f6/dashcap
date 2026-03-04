# .devcontainer/Dockerfile
FROM mcr.microsoft.com/devcontainers/go:1.25-trixie

RUN apt-get update && apt-get install -y \
    libpcap-dev \
  && rm -rf /var/lib/apt/lists/*