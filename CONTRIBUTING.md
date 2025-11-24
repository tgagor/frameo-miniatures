# Contributing to Frameo Miniatures

Thank you for your interest in contributing to Frameo Miniatures! This document provides guidelines and information for contributors.

## Development Setup

### Prerequisites

- Go 1.25.4 or later
- Make
- Git

### Clone and Build

```bash
git clone https://github.com/tgagor/frameo-miniatures.git
cd frameo-miniatures
make build
```

## Development Workflow

### Building

Build the binary:

```bash
make build
```

The binary will be created at `bin/frameo-miniatures`.

### Running

Run directly from source:

```bash
make run
```

Or run the built binary:

```bash
./bin/frameo-miniatures -i ./example -o ./output
```

### Cleaning

Remove build artifacts and test directories:

```bash
make clean
```

## Testing

### Unit Tests

Run Go unit tests:

```bash
make test
```

This runs all unit tests in the codebase with verbose output.

### Integration Tests

Run comprehensive integration tests:

```bash
make integration-test
```

This will execute the following test scenarios:

1. **Version Check** - Verifies the binary runs and displays version information
2. **Basic WebP Processing** - Default behavior with WebP output format
3. **JPEG Output Format** - Tests `-f jpg -q 85` flags
4. **Custom Resolution** - Tests `-r 1920x1080` flag
5. **Dry-Run Mode** - Verifies `--dry-run` doesn't create files
6. **Skip Existing Files** - Tests `--skip-existing` flag for incremental updates
7. **Prune Orphaned Files** - Tests `--prune` flag removes orphaned files
8. **Combined Flags** - Tests `--skip-existing --prune` together

#### Test Implementation

Integration tests are implemented as Makefile targets. Each test:
- Uses the `example/` directory as input
- Creates output in `test-output/` subdirectories
- Verifies expected behavior with assertions
- Cleans up automatically after completion

#### Running Individual Tests

You can run individual integration tests:

```bash
make test-version
make test-basic-webp
make test-jpeg-output
make test-custom-resolution
make test-dry-run
make test-skip-existing
make test-prune
make test-skip-existing-prune
```

## Code Quality

### Linting

Install linters:

```bash
make install-linters
```

Run all linters:

```bash
make lint
```

This runs pre-commit hooks which include:
- `goimports` - Format imports
- `gocyclo` - Check cyclomatic complexity
- `golangci-lint` - Comprehensive Go linter
- `gocritic` - Go source code analyzer

### Pre-commit Hooks

The project uses pre-commit hooks. Install them with:

```bash
pre-commit install
```

## Pull Request Process

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following the code style
3. **Add tests** for new functionality
4. **Run tests** to ensure everything passes:
   ```bash
   make test
   make integration-test
   ```
5. **Run linters** to ensure code quality:
   ```bash
   make lint
   ```
6. **Commit your changes** with clear, descriptive commit messages
7. **Push to your fork** and submit a pull request

## Commit Message Guidelines

- Use clear, descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Keep the first line under 72 characters
- Add detailed description if needed after a blank line

Examples:
```
Add support for PNG output format

Fix EXIF date preservation for HEIC files

Update documentation for --prune flag
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting (automatically done by linters)
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions focused and small

## Adding New Features

When adding new features:

1. **Discuss first** - Open an issue to discuss the feature before implementing
2. **Update documentation** - Update README.md with new flags/features
3. **Add tests** - Include both unit and integration tests
4. **Update examples** - Add examples if applicable

## Reporting Bugs

When reporting bugs, please include:

- Go version (`go version`)
- Operating system and version
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Any error messages or logs

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration:

- **Build** - Builds binaries for Linux, macOS, and Windows
- **Pre-commit** - Runs linters and code quality checks
- **Unit Tests** - Runs Go unit tests
- **Integration Tests** - Runs integration tests on all platforms
- **Release** - Automatically creates releases on main branch

All checks must pass before a PR can be merged.

## Project Structure

```
frameo-miniatures/
├── cmd/              # Command-line interface
├── internal/         # Internal packages
│   ├── app/         # Main application logic
│   ├── config/      # Configuration handling
│   ├── converter/   # Image conversion
│   ├── discovery/   # File discovery
│   ├── ignore/      # Ignore pattern handling
│   ├── processor/   # Image processing
│   ├── pruner/      # File pruning logic
│   └── ...
├── example/         # Example images for testing
├── test/            # Test files
├── Makefile         # Build and test automation
└── README.md        # User documentation
```

## Questions?

If you have questions about contributing, feel free to:
- Open an issue for discussion
- Ask in pull request comments
- Check existing issues and PRs for similar questions

## License

By contributing to Frameo Miniatures, you agree that your contributions will be licensed under the same license as the project.

Thank you for contributing! 🎉
