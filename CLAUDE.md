# Redis Export - Claude Development Context

## Project Overview

This is a high-performance Redis database exporter written in Go that exports all keys and values from a Redis database to JSON format. The tool was built with concurrent processing capabilities to handle large Redis datasets efficiently.

## Architecture

The exporter follows a concurrent worker pool pattern:

1. **Key Scanning**: Uses Redis SCAN to iterate through all keys without blocking
2. **Worker Pool**: Configurable number of goroutines process keys in parallel
3. **Type Detection**: Automatically detects and handles all Redis data types
4. **JSON Streaming**: Efficiently streams results to JSON file
5. **Structured Logging**: Uses logrus for configurable logging with performance metrics

## Key Components

### Core Files
- `main.go`: Main application with CLI interface using Cobra
- `main_test.go`: Unit tests for core functionality
- `exporter_test.go`: Integration tests with Redis mocks

### Supported Redis Data Types
- `string`: Simple key-value pairs
- `list`: Ordered collections
- `set`: Unordered unique collections  
- `zset`: Sorted sets with scores
- `hash`: Field-value maps
- `stream`: Event streams

### Configuration
- Redis connection (address, password, database)
- Output file path
- Worker concurrency (defaults to 2x CPU cores)
- Batch size for key scanning
- Log level (trace, debug, info, warn, error, fatal, panic)
- TTL preservation for expiring keys

## Development Commands

### Building
```bash
go build -o redis-export
```

### Testing
```bash
go test -v                    # Run all tests
go test -v -race              # Run with race detection
go test -v -cover             # Run with coverage
```

### Linting
```bash
golangci-lint run            # Run linter
```

### Cross-platform Build
```bash
GOOS=linux GOARCH=amd64 go build -o redis-export-linux
GOOS=windows GOARCH=amd64 go build -o redis-export.exe
GOOS=darwin GOARCH=arm64 go build -o redis-export-mac
```

## Dependencies

- `github.com/redis/go-redis/v9`: Redis client library
- `github.com/spf13/cobra`: CLI framework
- `github.com/sirupsen/logrus`: Structured logging library
- `github.com/stretchr/testify`: Testing assertions
- `github.com/go-redis/redismock/v9`: Redis mocking for tests

## CI/CD Pipeline

The project uses GitHub Actions with three workflows:

### CI (`ci.yml`)
- Multi-version Go testing (1.21, 1.22, 1.23)
- Code linting and security scanning
- Multi-platform builds
- Test coverage reporting

### Release (`release.yml`)
- Triggered on semantic version tags (v0.1.0, v1.2.3)
- Cross-platform binary compilation
- Automatic changelog generation
- GitHub Releases with downloadable assets

### Tag Creation (`tag-release.yml`)
- Manual workflow for creating release tags
- Semantic version validation
- Automatic release pipeline trigger

## Performance Considerations

1. **Concurrent Processing**: Uses worker pools to parallelize key processing
2. **Buffered Channels**: Prevents blocking between scanning and processing
3. **Streaming JSON**: Avoids loading entire dataset into memory
4. **Batch Scanning**: Configurable batch sizes for optimal Redis performance
5. **Context Cancellation**: Proper cleanup on interruption

## Testing Strategy

- **Unit Tests**: Individual component testing with mocks
- **Integration Tests**: End-to-end testing with mock Redis
- **Race Detection**: Concurrent safety verification
- **Error Scenarios**: Network failures, invalid paths, cancelled contexts

## Error Handling

- Connection failures are logged and cause immediate exit
- Individual key processing errors are logged but don't stop export
- File I/O errors cause immediate failure
- Context cancellation is handled gracefully

## Future Enhancements

Potential improvements could include:
- Pattern-based key filtering
- Incremental exports with timestamps
- Multiple output formats (CSV, XML)
- Compression options
- Memory usage optimization for very large datasets
- Redis Cluster support
- Backup restoration capabilities

## Troubleshooting

Common issues:
1. **Connection timeouts**: Check Redis connectivity and credentials
2. **Large dataset performance**: Adjust worker count and batch size
3. **Memory usage**: Monitor for very large values in Redis
4. **File permissions**: Ensure write access to output directory

## Code Style

The project follows standard Go conventions:
- gofmt formatting
- golint compliance
- Proper error handling
- Clear variable naming
- Comprehensive documentation