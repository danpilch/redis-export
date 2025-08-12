# Redis Export

A high-performance Redis database exporter that exports all keys and values from a Redis database to JSON format with concurrent processing.

## Features

- **High Performance**: Concurrent worker pools for parallel key processing
- **All Redis Data Types**: Supports string, list, set, zset, hash, and stream types
- **TTL Preservation**: Maintains expiration information for keys
- **Progress Reporting**: Real-time progress updates with structured logging
- **Structured Logging**: Configurable log levels with detailed performance metrics
- **Cross-Platform**: Binaries available for Linux, macOS, and Windows
- **Configurable**: Adjustable concurrency, batch sizes, and connection parameters
- **Safe Operation**: Uses Redis SCAN to avoid blocking the server

## Installation

### Download Pre-built Binaries

Download the latest release from the [GitHub Releases](https://github.com/your-username/redis-export/releases) page.

Available platforms:
- Linux (x86_64, ARM64)
- macOS (x86_64, ARM64) 
- Windows (x86_64)

### Build from Source

Requirements:
- Go 1.21 or later

```bash
git clone https://github.com/your-username/redis-export.git
cd redis-export
go build -o redis-export
```

### Install via Go

```bash
go install github.com/your-username/redis-export@latest
```

## Quick Start

```bash
# Export from local Redis to JSON file
./redis-export -a localhost:6379 -o my-redis-export.json

# Export from remote Redis with authentication
./redis-export -a redis.example.com:6379 -p mypassword -o export.json

# Export with custom concurrency and logging
./redis-export -a localhost:6379 -w 16 -b 2000 -o export.json --log-level debug
```

## Usage

```
redis-export [flags]

Flags:
  -a, --addr string        Redis server address (default "localhost:6379")
  -b, --batch int          Batch size for key scanning (default 1000)
  -d, --db int             Redis database number (default 0)
  -h, --help               Help for redis-export
  -l, --log-level string   Log level (trace, debug, info, warn, error, fatal, panic) (default "info")
  -o, --output string      Output JSON file (default "redis_export.json")
  -p, --password string    Redis password
  -w, --workers int        Number of worker goroutines (default: 2x CPU cores)
  -v, --version            Show version information
```

## Examples

### Basic Export

Export all data from a local Redis instance:

```bash
./redis-export -a localhost:6379 -o backup.json
```

### Remote Redis with Authentication

Export from a remote Redis server with password:

```bash
./redis-export \
  -a redis.example.com:6379 \
  -p "your-password" \
  -d 1 \
  -o production-backup.json
```

### High-Performance Export

Export with increased concurrency for large datasets:

```bash
./redis-export \
  -a localhost:6379 \
  -w 32 \
  -b 5000 \
  -o large-export.json \
  --log-level info
```

### Debug Export with Verbose Logging

Export with detailed logging for troubleshooting:

```bash
./redis-export \
  -a localhost:6379 \
  -o debug-export.json \
  --log-level debug
```

### Docker Usage

```bash
# Using pre-built image
docker run -v $(pwd):/output \
  your-username/redis-export:latest \
  -a host.docker.internal:6379 \
  -o /output/export.json

# Build and run locally
docker build -t redis-export .
docker run -v $(pwd):/output redis-export \
  -a your-redis-host:6379 -o /output/backup.json
```

## Output Format

The exporter generates a JSON array with each Redis key as an object:

```json
[
  {
    "key": "user:1001",
    "type": "hash",
    "value": {
      "name": "John Doe",
      "email": "john@example.com",
      "age": "30"
    },
    "ttl": 3600
  },
  {
    "key": "session:abc123",
    "type": "string", 
    "value": "user-session-data"
  },
  {
    "key": "leaderboard",
    "type": "zset",
    "value": [
      {"Score": 1000, "Member": "player1"},
      {"Score": 950, "Member": "player2"}
    ]
  }
]
```

### Field Descriptions

- `key`: The Redis key name
- `type`: Redis data type (string, list, set, zset, hash, stream)
- `value`: The actual data (format varies by type)
- `ttl`: Time-to-live in seconds (omitted for persistent keys)

## Performance Tuning

### Worker Threads

The `-w` flag controls concurrent workers. General guidelines:

- **Small datasets**: Use default (2x CPU cores)
- **Large datasets**: Increase to 16-32 workers
- **Network-bound**: Higher worker counts help
- **CPU-bound**: Don't exceed 2-4x CPU cores

### Batch Size

The `-b` flag controls how many keys are fetched per SCAN operation:

- **Small values**: Use 1000-2000 (default)
- **Large values**: Decrease to 500-1000
- **Fast network**: Increase to 5000-10000
- **Slow network**: Decrease to 100-500

### Memory Considerations

- Large Redis values will temporarily consume memory during processing
- Workers process keys concurrently, multiplying memory usage
- Monitor memory usage and adjust worker count if needed

## Data Type Handling

| Redis Type | Export Format | Notes |
|------------|---------------|-------|
| String | `"value"` | Direct string value |
| List | `["item1", "item2"]` | Array of strings |
| Set | `["member1", "member2"]` | Array of unique strings |
| ZSet | `[{"Score": 1.0, "Member": "item"}]` | Array of score-member objects |
| Hash | `{"field1": "value1"}` | Object with field-value pairs |
| Stream | `[{"ID": "1-0", "Values": {...}}]` | Array of stream entries |

## Error Handling

The exporter handles various error conditions:

- **Connection failures**: Immediate exit with error message
- **Individual key errors**: Logged but export continues
- **File write errors**: Immediate exit with error message
- **Interrupted exports**: Graceful shutdown with partial results

## Monitoring

The tool provides structured logging with configurable verbosity levels:

### Log Levels
- **trace**: Extremely detailed debugging information
- **debug**: Detailed debugging information  
- **info**: General operational messages (default)
- **warn**: Warning messages
- **error**: Error messages only
- **fatal/panic**: Critical errors

### Example Output (--log-level info):
```
time="2025-08-12T10:30:00+01:00" level=info msg="Connecting to Redis" redis_addr="localhost:6379"
time="2025-08-12T10:30:00+01:00" level=info msg="Successfully connected to Redis" response="PONG"
time="2025-08-12T10:30:00+01:00" level=info msg="Starting Redis export" batch_size=1000 output_file="backup.json" workers=24
time="2025-08-12T10:30:05+01:00" level=info msg="Export progress" elapsed=5s keys_per_sec=7234.5 processed_keys=36172
time="2025-08-12T10:30:10+01:00" level=info msg="Export progress" elapsed=10s keys_per_sec=7156.3 processed_keys=71563
time="2025-08-12T10:30:15+01:00" level=info msg="Export completed successfully" avg_keys_per_sec=7198.6 total_duration=15s total_keys=107979
```

### Error Logging:
```
time="2025-08-12T10:30:05+01:00" level=error msg="Error processing key: connection timeout" key="large:dataset:key123"
```

## Troubleshooting

### Common Issues

**Connection Timeout**
```
Error: failed to connect to Redis: dial tcp: i/o timeout
```
- Check Redis server address and port
- Verify firewall settings
- Test with `redis-cli ping`

**Authentication Failed**
```
Error: failed to connect to Redis: AUTH failed
```
- Verify password with `-p` flag
- Check Redis AUTH configuration

**Permission Denied**
```
Error: failed to create output file: permission denied
```
- Check write permissions for output directory
- Use absolute path or different location

**Large Dataset Performance**
- Increase worker count: `-w 32`
- Adjust batch size: `-b 2000`
- Monitor system resources
- Use `--log-level info` to track progress

### Performance Issues

**Slow Export Speed**
1. Increase workers: `-w 16`
2. Increase batch size: `-b 5000`
3. Check network latency to Redis
4. Monitor Redis server load
5. Use `--log-level debug` for detailed performance analysis

**High Memory Usage**
1. Decrease worker count: `-w 4`
2. Decrease batch size: `-b 500`
3. Check for very large Redis values
4. Use `--log-level error` to reduce logging overhead

**Troubleshooting Connection Issues**
- Use `--log-level debug` to see detailed connection information
- Check Redis server logs for authentication or network issues
- Verify Redis configuration allows external connections

## Development

See [CLAUDE.md](CLAUDE.md) for development setup, testing, and contribution guidelines.

### Building

```bash
# Build for current platform
go build -o redis-export

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o redis-export-linux
GOOS=windows GOARCH=amd64 go build -o redis-export.exe
GOOS=darwin GOARCH=arm64 go build -o redis-export-mac
```

### Testing

```bash
go test -v                 # Run tests
go test -v -race          # Run with race detection
go test -v -cover         # Run with coverage
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Support

- Report bugs and request features via [GitHub Issues](https://github.com/your-username/redis-export/issues)
- For questions, start a [GitHub Discussion](https://github.com/your-username/redis-export/discussions)

## Changelog

See [Releases](https://github.com/your-username/redis-export/releases) for version history and changes.