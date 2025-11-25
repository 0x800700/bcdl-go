# Building Bandcamp Downloader from Source

This guide provides detailed instructions for building the Bandcamp Downloader application from source code on macOS.

## ⚠️ Before You Start

**Do you really need to build from source?**

For most users, we **strongly recommend** downloading the pre-built application from [GitHub Releases](https://github.com/0x800700/bcdl-go/releases). Building from source requires:

- ~3 GB of development tools
- 30-60 minutes of setup time
- Technical knowledge of command-line tools
- Modifications to your system PATH

**Only proceed if you:**
- Want to contribute to development
- Need to modify the source code
- Understand the implications of installing development tools

## 📋 System Requirements

- **macOS**: 10.15 (Catalina) or later
- **Disk Space**: ~5 GB free (for tools + build artifacts)
- **RAM**: 8 GB minimum, 16 GB recommended
- **Internet**: Required for downloading dependencies

## 🛠️ Step 1: Install Xcode Command Line Tools

Xcode Command Line Tools provide essential compilers and build tools.

```bash
xcode-select --install
```

A dialog will appear. Click "Install" and wait for completion (~10-20 minutes).

**Verify installation:**
```bash
xcode-select -p
# Should output: /Applications/Xcode.app/Contents/Developer
# or: /Library/Developer/CommandLineTools
```

**Size**: ~1.5 GB

## 🐹 Step 2: Install Go

Go is required to compile the backend.

### Option A: Official Installer (Recommended)

1. Visit [https://golang.org/dl/](https://golang.org/dl/)
2. Download the macOS installer (`.pkg` file)
3. Run the installer
4. Follow the installation wizard

### Option B: Homebrew

```bash
brew install go
```

**Verify installation:**
```bash
go version
# Should output: go version go1.23.x darwin/amd64 (or darwin/arm64)
```

**Size**: ~150 MB

## 📦 Step 3: Install Node.js

Node.js is required to build the frontend.

### Option A: Official Installer (Recommended)

1. Visit [https://nodejs.org/](https://nodejs.org/)
2. Download the LTS version for macOS
3. Run the installer
4. Follow the installation wizard

### Option B: Homebrew

```bash
brew install node
```

**Verify installation:**
```bash
node --version  # Should be v20.x or higher
npm --version   # Should be v10.x or higher
```

**Size**: ~50 MB

## 🚀 Step 4: Install Wails CLI

Wails is the framework that combines Go and the web frontend.

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

This installs Wails to `~/go/bin/wails`.

**Verify installation:**
```bash
~/go/bin/wails version
# Should output: Wails CLI v2.11.0
```

**Size**: ~30 MB

## 📥 Step 5: Clone the Repository

```bash
# Clone the repository
git clone https://github.com/0x800700/bcdl-go.git
cd bcdl-go

# Install Go dependencies
go mod download
```

## 🔨 Step 6: Build the Application

```bash
# Build for macOS (Universal binary - Intel + Apple Silicon)
~/go/bin/wails build -platform darwin/universal -clean
```

**Build process:**
1. Generates Go ↔ JavaScript bindings
2. Installs npm dependencies (~200 MB in `frontend/node_modules/`)
3. Compiles React frontend with Vite
4. Compiles Go backend
5. Downloads Playwright browsers (~1 GB, cached in `~/Library/Caches/ms-playwright/`)
6. Packages everything into `.app` bundle

**Time**: 2-5 minutes (first build), 30-60 seconds (subsequent builds)

## 📂 Step 7: Locate the Built Application

The compiled application will be at:

```
build/bin/Bandcamp Downloader.app
```

**Size**: ~1.1 GB (includes embedded Chromium browser)

You can now:
- Double-click to run
- Move to `/Applications/`
- Distribute to others (macOS only)

## 🧹 Cleaning Up Build Artifacts

To save disk space after building:

```bash
# Remove build artifacts (keeps source code)
rm -rf build/bin/
rm -rf frontend/node_modules/
rm -rf frontend/dist/

# Remove Playwright browser cache (can be re-downloaded)
rm -rf ~/Library/Caches/ms-playwright/
```

## 🔧 Development Mode

For active development with hot-reload:

```bash
~/go/bin/wails dev
```

This will:
- Start a development server
- Open the app with live reload
- Show console logs in terminal
- Rebuild on file changes

## ❓ Troubleshooting

### "wails: command not found"

The Wails binary is in `~/go/bin/`, which may not be in your PATH.

**Solution**: Use the full path `~/go/bin/wails` or add to PATH:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### "xcode-select: error: tool 'xcodebuild' requires Xcode"

You need Command Line Tools, not full Xcode.

**Solution**: Run `xcode-select --install` again.

### "npm ERR! network timeout"

Network issue downloading npm packages.

**Solution**: Try again with increased timeout:

```bash
cd frontend
npm install --timeout=60000
cd ..
```

### Build fails with "playwright not found"

Playwright browsers not downloaded.

**Solution**: The first build automatically downloads browsers. If it fails:

```bash
cd frontend
npx playwright install chromium
cd ..
```

### "Permission denied" errors

Incorrect file permissions.

**Solution**:
```bash
chmod +x ~/go/bin/wails
```

## 🗑️ Uninstalling Development Tools

If you want to remove all development tools after building:

### Remove Wails
```bash
rm ~/go/bin/wails
```

### Remove Go
```bash
# If installed via official installer:
sudo rm -rf /usr/local/go
sudo rm /etc/paths.d/go

# If installed via Homebrew:
brew uninstall go
```

### Remove Node.js
```bash
# If installed via official installer:
# Use the official uninstaller or:
sudo rm -rf /usr/local/lib/node_modules
sudo rm /usr/local/bin/node
sudo rm /usr/local/bin/npm

# If installed via Homebrew:
brew uninstall node
```

### Remove Xcode Command Line Tools
```bash
sudo rm -rf /Library/Developer/CommandLineTools
```

## 📚 Additional Resources

- [Wails Documentation](https://wails.io/docs/introduction)
- [Go Documentation](https://golang.org/doc/)
- [Node.js Documentation](https://nodejs.org/docs/)
- [Playwright Documentation](https://playwright.dev/)

## 🆘 Getting Help

If you encounter issues:

1. Check the [Issues](https://github.com/0x800700/bcdl-go/issues) page
2. Search for similar problems
3. Create a new issue with:
   - Your macOS version
   - Go version (`go version`)
   - Node version (`node --version`)
   - Full error message
   - Steps to reproduce

---

**Remember**: For most users, downloading the pre-built app from [Releases](https://github.com/0x800700/bcdl-go/releases) is much easier!
