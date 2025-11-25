#!/bin/bash
# Quick guide to push to GitHub

echo "📤 Pushing Bandcamp Downloader to GitHub"
echo "========================================="
echo ""

# Check git status
echo "📊 Current git status:"
git status --short
echo ""

# Confirm with user
read -p "Ready to commit and push? (y/n) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    echo "❌ Aborted"
    exit 1
fi

# Add files
echo "➕ Adding files..."
git add README.md BUILDING.md CONTRIBUTING.md .gitignore docs/ scripts/

# Commit
echo "💾 Committing..."
git commit -m "docs: Add comprehensive documentation for v2.0.0

- Professional README with features, architecture, and screenshots
- Detailed BUILDING.md with step-by-step instructions
- CONTRIBUTING.md for developers
- PUBLISHING.md guide for GitHub releases
- Release packaging script
- Updated .gitignore to exclude build artifacts"

# Push
echo "🚀 Pushing to GitHub..."
git push origin main

echo ""
echo "✅ Successfully pushed to GitHub!"
echo ""
echo "📋 Next steps:"
echo "1. Visit: https://github.com/0x800700/bcdl-go"
echo "2. Verify files are uploaded"
echo "3. Create a release following docs/PUBLISHING.md"
echo ""
