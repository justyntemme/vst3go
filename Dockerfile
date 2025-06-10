FROM ubuntu:latest

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV CGO_ENABLED=1

# Enable 32-bit architecture
RUN dpkg --add-architecture i386

# Update package lists
RUN apt-get update

# Install required development packages
RUN apt-get install -y \
    # Build essentials
    build-essential \
    gcc \
    g++ \
    make \
    pkg-config \
    # 32-bit support
    gcc-multilib \
    g++-multilib \
    libc6-dev-i386 \
    # Go language
    golang-go \
    # VST3 development requirements
    cmake \
    ninja-build \
    # Audio development libraries (64-bit)
    libasound2-dev \
    libjack-jackd2-dev \
    # X11 development (64-bit)
    libx11-dev \
    libxext-dev \
    libxrandr-dev \
    libxcursor-dev \
    libxi-dev \
    libxinerama-dev \
    libxxf86vm-dev \
    libxss-dev \
    libgl1-mesa-dev \
    libglu1-mesa-dev \
    # Additional tools
    git \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install 32-bit libraries separately (only runtime libs, not dev packages)
RUN apt-get update && apt-get install -y \
    libc6:i386 \
    libstdc++6:i386 \
    libasound2:i386 \
    libjack-jackd2-0:i386 \
    libx11-6:i386 \
    libxext6:i386 \
    libgl1:i386 \
    && rm -rf /var/lib/apt/lists/*

# Create working directory
WORKDIR /workspace

# Copy the entire project
COPY . .

# Download Go dependencies
RUN go mod download

# Create build output directory
RUN mkdir -p /workspace/build

# Set the default command to build all plugins (both 32-bit and 64-bit)
CMD ["make", "build"]

# The build output will be available in ./build when using bind mount:
# docker run -v $(pwd)/build:/workspace/build vst3go-builder
#
# To build only 64-bit:
# docker run -v $(pwd)/build:/workspace/build vst3go-builder make build-64
#
# To build only 32-bit:
# docker run -v $(pwd)/build:/workspace/build vst3go-builder make build-32