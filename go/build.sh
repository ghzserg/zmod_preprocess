#!/bin/bash

APP_NAME="zmod_preprocess"
BUILD_DIR="dist"

mkdir -p $BUILD_DIR

echo "🛠️  Кросскомпиляция $APP_NAME..."

# Linux 64-bit
echo "📦 Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/${APP_NAME}_linux_amd64 main.go

# Windows 64-bit
echo "📦 Windows amd64..."
GOOS=windows GOARCH=amd64 go build -o $BUILD_DIR/${APP_NAME}_windows_amd64.exe main.go

# macOS Intel
echo "📦 macOS amd64..."
GOOS=darwin GOARCH=amd64 go build -o $BUILD_DIR/${APP_NAME}_darwin_amd64 main.go

# macOS Apple Silicon
echo "📦 macOS arm64..."
GOOS=darwin GOARCH=arm64 go build -o $BUILD_DIR/${APP_NAME}_darwin_arm64 main.go

echo ""
echo "✅ Сборка завершена!"
echo "📁 Файлы в папке: $BUILD_DIR/"
ls -lh $BUILD_DIR/