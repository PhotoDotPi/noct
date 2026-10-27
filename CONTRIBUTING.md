# Contributing to noct

Thanks for your interest in contributing! Every bit helps make noct better.

## Getting Started

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/<your-username>/noct.git`
3. **Create a branch**: `git checkout -b feature/your-feature`
4. **Make changes** and test them
5. **Commit** with clear messages: `git commit -m "feat: add something cool"`
6. **Push** to your fork: `git push origin feature/your-feature`
7. **Open a Pull Request**

## Development

### Prerequisites
- [Go 1.27+](https://go.dev)
- [mpv](https://mpv.io/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)

### Build & Run
```bash
go build ./cmd/noct
./noct
```

### Testing
```bash
go test ./...
```

## Guidelines

- **Keep it minimal** — noct is about simplicity and focus
- **Follow Go conventions** — `gofmt`, idiomatic code
- **UI consistency** — Match the existing theme in `internal/ui/`
- **Commit style** — Use [Conventional Commits](https://www.conventionalcommits.org/) where possible (`feat:`, `fix:`, `docs:`, `refactor:`, etc.)
- **Small PRs** — Easier to review and faster to merge

## Issues

- **Bugs**: Include steps to reproduce, OS, terminal, and logs if possible
- **Features**: Describe the use case and how it fits noct's minimal philosophy

## Code of Conduct

By participating, you agree to follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contributions will be licensed under the [GNU AGPLv3 License](LICENSE).
