# Configuration Precedence Example

This example demonstrates go-snap's powerful configuration system with automatic precedence handling.

## Features Demonstrated

### 🏷️ **Modern Struct Tags**
```go
type ServerConfig struct {
    Host string `flag:"host" env:"HOST" description:"Server hostname" default:"localhost"`
    Port int    `flag:"port,required" env:"PORT" description:"Server port" default:"8080"`

    // Ignored field - won't generate CLI flag
    Internal string `flag:",ignore"`

    // Enum validation
    LogLevel string `flag:"log-level" env:"LOG_LEVEL" enum:"debug,info,warn,error" default:"info"`

    // Nested groups
    Database struct {
        URL string `flag:"db-url,required" env:"DATABASE_URL" description:"Database URL"`
    } `group:"database"`
}
```

### 📊 **Configuration Precedence**
1. **Command line flags** (highest priority)
2. **Environment variables**
3. **Configuration files**
4. **Default values** (lowest priority)

### 🎯 **Advanced Features**
- **Zero-allocation parsing** for performance
- **Automatic CLI flag generation** from struct tags
- **Type-safe flag validation** with enums
- **Nested configuration groups**
- **Multiple data sources** with smart precedence
- **Enhanced duration parsing** (`30s`, `1h30m`, `01:30`)

## Run

### Basic Usage
```bash
# Use defaults and environment variables
export HOST=prod-server
export DEBUG=true
go run main.go
```

### CLI Flag Override
```bash
# CLI flags override environment variables
export HOST=env-server
go run main.go --host=cli-server --port=3000 --debug
# Result: host=cli-server (CLI wins over env)
```

### Database Configuration
```bash
# Configure database with required URL
go run main.go --db-url="postgres://user:pass@localhost/mydb" --db-max-conns=50
```

### Cache Configuration
```bash
# Configure Redis cache with nested groups
go run main.go --cache-type=redis --redis-host=cache.example.com --redis-port=6380
```

### Interactive Demo
```bash
# Run the interactive demo to see precedence in action
go run main.go demo
```

## Generated CLI Help

When you run with `--help`, go-snap automatically generates comprehensive help:

```
High-performance web server with advanced configuration

Usage:
  myserver [GLOBAL FLAGS]

database - Database configuration:
  --db-url value          Database connection URL
  --db-max-conns value    Maximum database connections (default: 20)
  --db-timeout value      Database connection timeout (default: 10s)
  --db-ssl                Enable SSL for database (default: true)
  --db-migrate            Run auto-migration on startup

cache - Cache configuration:
  --cache-type value      Cache backend type (valid values: redis, memory) (default: memory)
  --cache-ttl value       Default cache TTL (default: 1h)

redis - Redis configuration:
  --redis-host value      Redis hostname (default: localhost)
  --redis-port value      Redis port (default: 6379)
  --redis-pass value      Redis password
  --redis-db value        Redis database number (default: 0)

Global Flags:
  --host value            Server hostname (default: localhost)
  --port value            Server port (default: 8080)
  --debug                 Enable debug logging
  --timeout value         Request timeout (default: 30s)
  --workers value         Number of worker threads (default: 4)
  --log-level value       Logging level (valid values: debug, info, warn, error) (default: info)
  --allowed-ips value     Allowed IP addresses
```

## Key Benefits

✅ **Zero Boilerplate** - No manual flag definitions needed
✅ **Type Safety** - Automatic validation and conversion
✅ **Performance** - Zero-allocation parsing
✅ **Flexibility** - Multiple configuration sources
✅ **Developer Experience** - Excellent error messages and help

This example showcases why go-snap is the next-generation CLI library for Go!
