#!/bin/sh
set -eu

# Optional first argument selects the Go executable.
go_bin=${1:-go}
if [ "$#" -gt 1 ]; then
    echo "Usage: $0 [go-executable]" >&2
    exit 1
fi
if ! command -v "$go_bin" >/dev/null 2>&1; then
    echo "Go executable not found: $go_bin" >&2
    exit 1
fi

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$project_dir"

output_dir=bin/linux-amd64
mkdir -p "$output_dir/docs"

# Pure Go binaries avoid a dependency on the target Ubuntu glibc version.
export GOOS=linux GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0

for app in migrate backend; do
    echo "Building $app for linux/amd64..."
    "$go_bin" build -mod=readonly -trimpath -ldflags='-s -w' \
        -o "$output_dir/$app" "./cmd/$app"
done

cp docs/openapi.yaml "$output_dir/docs/openapi.yaml"
cp .env.example "$output_dir/.env.example"

echo "Build complete: $project_dir/$output_dir"
echo "Configure .env on the server, then run ./migrate before ./backend from that directory."
