#!/bin/bash
# Release preparation script for Bandcamp Downloader
# This script packages the built application for GitHub release

set -e

VERSION=${1:-"2.0.0"}
APP_NAME="Bandcamp Downloader"
OUTPUT_DIR="release"
APP_PATH="build/bin/${APP_NAME}.app"
ZIP_NAME="Bandcamp.Downloader.v${VERSION}.macOS.zip"

echo "🎵 Bandcamp Downloader - Release Preparation"
echo "=============================================="
echo ""

# Check if app exists
if [ ! -d "$APP_PATH" ]; then
    echo "❌ Error: Application not found at $APP_PATH"
    echo "Please build the application first:"
    echo "  ~/go/bin/wails build -platform darwin/universal -clean"
    exit 1
fi

# Get app size
APP_SIZE=$(du -sh "$APP_PATH" | cut -f1)
echo "📦 Application size: $APP_SIZE"
echo ""

# Create release directory
mkdir -p "$OUTPUT_DIR"

# Create ZIP archive
echo "📦 Creating ZIP archive..."
cd "build/bin"
zip -r -q "../../${OUTPUT_DIR}/${ZIP_NAME}" "${APP_NAME}.app"
cd ../..

ZIP_SIZE=$(du -sh "${OUTPUT_DIR}/${ZIP_NAME}" | cut -f1)
echo "✅ Created: ${OUTPUT_DIR}/${ZIP_NAME} (${ZIP_SIZE})"
echo ""

# Calculate SHA256
echo "🔐 Calculating SHA256 checksum..."
shasum -a 256 "${OUTPUT_DIR}/${ZIP_NAME}" > "${OUTPUT_DIR}/${ZIP_NAME}.sha256"
CHECKSUM=$(cat "${OUTPUT_DIR}/${ZIP_NAME}.sha256" | cut -d' ' -f1)
echo "✅ SHA256: $CHECKSUM"
echo ""

# Create release notes template
RELEASE_NOTES="${OUTPUT_DIR}/RELEASE_NOTES_v${VERSION}.md"
cat > "$RELEASE_NOTES" << EOF
# Bandcamp Downloader v${VERSION}

## 📥 Download

- **macOS (Universal)**: [${ZIP_NAME}](https://github.com/0x800700/bcdl-go/releases/download/v${VERSION}/${ZIP_NAME})
  - Size: ${ZIP_SIZE}
  - SHA256: \`${CHECKSUM}\`
  - Supports: Intel and Apple Silicon Macs

## ✨ Features

- 🔍 Smart artist scanning with real-time updates
- 💰 Support for free, NYP, and paid album detection
- 📧 Automated email verification for NYP albums
- 🎨 Modern dark-themed UI
- ⚡ Batch download support
- 🎼 Multiple format options (FLAC, MP3-320, etc.)

## 🚀 Installation

1. Download the ZIP file above
2. Extract the archive
3. Move \`Bandcamp Downloader.app\` to your Applications folder
4. Right-click and select "Open" (first time only, due to macOS Gatekeeper)
5. Enjoy!

## 📝 What's New in v${VERSION}

- Initial public release
- Full Bandcamp artist scanning
- Automated download workflows
- Temporary email integration

## 🐛 Known Issues

- Windows and Linux builds are not officially supported yet
- Large application size (~1.1 GB) due to embedded browser
- First launch may take a few seconds to initialize

## 🔮 Coming Soon

- Migration to Electron for smaller size and better cross-platform support
- Windows and Linux official support
- Download queue management
- Better error handling

## 📚 Documentation

- [README](https://github.com/0x800700/bcdl-go#readme)
- [Building from Source](https://github.com/0x800700/bcdl-go/blob/main/BUILDING.md)
- [Contributing](https://github.com/0x800700/bcdl-go/blob/main/CONTRIBUTING.md)

## 🙏 Acknowledgments

Built with [Wails](https://wails.io/), [Playwright](https://playwright.dev/), and [Mail.tm](https://mail.tm/).

---

**Full Changelog**: https://github.com/0x800700/bcdl-go/commits/v${VERSION}
EOF

echo "📝 Created release notes: $RELEASE_NOTES"
echo ""

echo "✅ Release package ready!"
echo ""
echo "📋 Next steps:"
echo "1. Review release notes: $RELEASE_NOTES"
echo "2. Create a new release on GitHub:"
echo "   https://github.com/0x800700/bcdl-go/releases/new"
echo "3. Tag: v${VERSION}"
echo "4. Upload: ${OUTPUT_DIR}/${ZIP_NAME}"
echo "5. Copy release notes from: $RELEASE_NOTES"
echo ""
echo "🎉 Done!"
