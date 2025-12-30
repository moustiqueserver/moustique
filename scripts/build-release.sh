#!/bin/bash
# Build Moustique binaries for all platforms

set -e

VERSION=${1:-$(git describe --tags --always)}
BUILD_DIR="build"

echo "Building Moustique v${VERSION} for all platforms..."
echo "=================================================="

# Clean build directory
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Build for different platforms
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS="${array[0]}"
    GOARCH="${array[1]}"

    output_name="moustique-${GOOS}-${GOARCH}"

    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building for ${GOOS}/${GOARCH}..."

    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.Version=${VERSION}" \
        -o "${BUILD_DIR}/${output_name}" \
        .

    echo "✓ Built ${BUILD_DIR}/${output_name}"
done

echo ""
echo "=================================================="
echo "Build complete! Binaries in ${BUILD_DIR}/"
ls -lh ${BUILD_DIR}/
