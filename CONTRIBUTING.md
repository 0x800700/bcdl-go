# Contributing to Bandcamp Downloader

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## 🚀 Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/bcdl-go.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Test thoroughly
6. Commit with clear messages
7. Push to your fork
8. Open a Pull Request

## 🏗️ Development Setup

See [BUILDING.md](BUILDING.md) for detailed setup instructions.

Quick start:
```bash
# Install dependencies
go mod download

# Run in development mode
~/go/bin/wails dev
```

## 📝 Code Style

### Go
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Add comments for exported functions
- Handle errors explicitly

### TypeScript/React
- Use TypeScript strict mode
- Follow React best practices
- Use functional components with hooks
- Keep components small and focused

## 🧪 Testing

Currently, the project does not have automated tests. Contributions to add testing infrastructure are welcome!

Manual testing checklist:
- [ ] Scan artist page successfully
- [ ] Download free album
- [ ] Download NYP album (set to $0)
- [ ] Handle paid albums correctly
- [ ] UI updates in real-time
- [ ] Error handling works

## 🐛 Reporting Bugs

When reporting bugs, please include:
- macOS version
- App version
- Steps to reproduce
- Expected behavior
- Actual behavior
- Screenshots if applicable
- Console logs (if available)

## 💡 Feature Requests

Feature requests are welcome! Please:
- Check existing issues first
- Describe the use case
- Explain why it would be useful
- Consider implementation complexity

## 🔮 Roadmap

Current priorities:
1. Migration to Electron (in progress)
2. Cross-platform support (Windows, Linux)
3. Automated testing
4. Better error handling
5. Download queue management

## 📄 License

By contributing, you agree that your contributions will be licensed under the same license as the project.

## 🙏 Thank You

Every contribution, no matter how small, is appreciated!
