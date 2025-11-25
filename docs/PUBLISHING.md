# 📤 Publishing to GitHub

This guide walks you through publishing the Bandcamp Downloader to GitHub.

## 📋 Prerequisites

- [ ] Application is built and tested
- [ ] Release package is created (run `./scripts/prepare-release.sh`)
- [ ] You have push access to the repository

## 🚀 Step-by-Step Guide

### 1. Prepare the Repository

```bash
# Make sure you're on the main branch
git checkout main

# Stage all documentation files
git add README.md BUILDING.md CONTRIBUTING.md .gitignore docs/

# Commit
git commit -m "docs: Add comprehensive documentation for v2.0.0"

# Push to GitHub
git push origin main
```

### 2. Create a GitHub Release

1. **Go to the Releases page:**
   - Visit: https://github.com/0x800700/bcdl-go/releases
   - Click "Create a new release" or "Draft a new release"

2. **Choose a tag:**
   - Click "Choose a tag"
   - Type: `v2.0.0`
   - Click "Create new tag: v2.0.0 on publish"

3. **Set release title:**
   ```
   Bandcamp Downloader v2.0.0
   ```

4. **Add release notes:**
   - Copy content from: `release/RELEASE_NOTES_v2.0.0.md`
   - Paste into the description field

5. **Upload the application:**
   - Click "Attach binaries by dropping them here or selecting them"
   - Select: `release/Bandcamp.Downloader.v2.0.0.macOS.zip`
   - Optionally also upload: `release/Bandcamp.Downloader.v2.0.0.macOS.zip.sha256`

6. **Publish:**
   - ✅ Check "Set as the latest release"
   - Click "Publish release"

### 3. Verify the Release

After publishing:

1. **Check the release page:**
   - https://github.com/0x800700/bcdl-go/releases/tag/v2.0.0
   - Verify the ZIP file is downloadable

2. **Test the download:**
   - Download the ZIP from the release page
   - Extract and verify the app runs

3. **Update README links:**
   - If needed, update any "Download" links in README.md to point to the release

## 📝 Release Checklist

Before publishing:

- [ ] All tests pass (manual testing)
- [ ] Documentation is up to date
- [ ] Version number is correct in `wails.json`
- [ ] Screenshot is included in `docs/`
- [ ] `.gitignore` excludes build artifacts
- [ ] Release notes are complete
- [ ] SHA256 checksum is generated

After publishing:

- [ ] Release is visible on GitHub
- [ ] Download link works
- [ ] Application runs when downloaded
- [ ] README links to the release

## 🔄 Updating an Existing Release

If you need to update a release:

1. Go to the release page
2. Click "Edit release"
3. Make changes
4. Click "Update release"

To replace the binary:
1. Delete the old ZIP file from the release
2. Upload the new ZIP file
3. Update the SHA256 in release notes

## 🏷️ Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **Major** (X.0.0): Breaking changes, major rewrites
- **Minor** (2.X.0): New features, backwards compatible
- **Patch** (2.0.X): Bug fixes, small improvements

Examples:
- `v2.0.0` - Initial public release
- `v2.1.0` - Add download queue feature
- `v2.0.1` - Fix scanning bug

## 🎯 Future Releases

For subsequent releases:

```bash
# Update version in wails.json
# Build the app
~/go/bin/wails build -platform darwin/universal -clean

# Create release package
./scripts/prepare-release.sh 2.1.0

# Follow steps above
```

## 📚 Additional Resources

- [GitHub Releases Documentation](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository)
- [Semantic Versioning](https://semver.org/)
- [Writing Good Release Notes](https://keepachangelog.com/)

---

**Ready to publish? Let's go! 🚀**
