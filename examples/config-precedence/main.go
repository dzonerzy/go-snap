package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dzonerzy/go-snap/snap"
)

// ServerConfig demonstrates real-world configuration with precedence
type ServerConfig struct {
	// Basic server settings
	Host string `flag:"host" env:"HOST" description:"Server hostname" default:"localhost"`
	Port int    `flag:"port" env:"PORT" description:"Server port" default:"8080"`

	// Feature flags and timeouts
	Debug   bool          `flag:"debug" env:"DEBUG" description:"Enable debug logging"`
	Timeout time.Duration `flag:"timeout" env:"TIMEOUT" description:"Request timeout" default:"30s"`
	Workers int           `flag:"workers" env:"WORKERS" description:"Number of worker threads" default:"4"`

	// Configuration with enum validation
	LogLevel string `flag:"log-level" env:"LOG_LEVEL" description:"Logging level" enum:"debug,info,warn,error" default:"info"`

	// Slice configuration
	AllowedIPs []string `flag:"allowed-ips" env:"ALLOWED_IPS" description:"Allowed IP addresses" default:"127.0.0.1,::1"`

	// Database configuration group
	Database struct {
		URL         string        `flag:"db-url" env:"DATABASE_URL" description:"Database connection URL" default:"sqlite://app.db"`
		MaxConns    int           `flag:"db-max-conns" env:"DB_MAX_CONNS" description:"Maximum database connections" default:"20"`
		Timeout     time.Duration `flag:"db-timeout" env:"DB_TIMEOUT" description:"Database connection timeout" default:"10s"`
		EnableSSL   bool          `flag:"db-ssl" env:"DB_SSL" description:"Enable SSL for database" default:"true"`
		AutoMigrate bool          `flag:"db-migrate" env:"DB_MIGRATE" description:"Run auto-migration on startup"`
	} `group:"database"`

	// Cache configuration group
	Cache struct {
		Type string `flag:"cache-type" env:"CACHE_TYPE" description:"Cache backend type" enum:"redis,memory" default:"memory"`
		TTL  string `flag:"cache-ttl" env:"CACHE_TTL" description:"Default cache TTL" default:"1h"`

		// Redis-specific config (nested group)
		Redis struct {
			Host     string `flag:"redis-host" env:"REDIS_HOST" description:"Redis hostname" default:"localhost"`
			Port     int    `flag:"redis-port" env:"REDIS_PORT" description:"Redis port" default:"6379"`
			Password string `flag:"redis-pass" env:"REDIS_PASSWORD" description:"Redis password"`
			Database int    `flag:"redis-db" env:"REDIS_DB" description:"Redis database number" default:"0"`
		} `group:"redis"`
	} `group:"cache"`

	// Metrics and monitoring
	Metrics struct {
		Enabled bool   `flag:"metrics" env:"METRICS_ENABLED" description:"Enable metrics collection" default:"true"`
		Port    int    `flag:"metrics-port" env:"METRICS_PORT" description:"Metrics server port" default:"9090"`
		Path    string `flag:"metrics-path" env:"METRICS_PATH" description:"Metrics endpoint path" default:"/metrics"`
	} `group:"metrics"`
}

func main() {
	var config ServerConfig

	// Create configuration builder with all sources
	app, err := snap.Config("webserver", "Production-ready web server with advanced configuration").
		FromDefaults(snap.D{
			"host":    "0.0.0.0", // Override default for production
			"workers": 8,         // More workers for production
		}).
		FromEnv().               // Load from environment variables
        FromFlags().             // Generate CLI flags and load from command line
        FromFile("config.json"). // Load from JSON config file if provided
		Bind(&config).           // Bind to struct
		Build()                  // Build returns App when FromFlags() is used

	if err != nil {
		log.Fatalf("❌ Configuration setup error: %v", err)
	}

	// Run the app (this will handle --help, parse flags, and populate config)
	if err := app.Run(); err != nil {
		// Check if it's a graceful exit (help/version shown)
		if err == snap.ErrHelpShown || err == snap.ErrVersionShown {
			// Graceful exit - help or version was shown
			return
		}
		log.Fatalf("❌ Application error: %v", err)
	}

	// Configuration is now populated, start the server
	startServer(config)
}

func startServer(config ServerConfig) {
	// Print startup configuration
	printStartupInfo(config)

	// Setup HTTP server
	mux := http.NewServeMux()

	// Add basic routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := fmt.Sprintf(`{
	"status": "ok",
	"server": "%s:%d",
	"debug": %v,
	"log_level": "%s",
	"workers": %d,
	"database": "%s",
	"cache": "%s",
	"allowed_ips": %v
}`, config.Host, config.Port, config.Debug, config.LogLevel,
			config.Workers, config.Database.URL, config.Cache.Type, config.AllowedIPs)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		response := fmt.Sprintf(`{
	"database": {
		"url": "%s",
		"max_connections": %d,
		"ssl_enabled": %v,
		"timeout": "%v"
	},
	"cache": {
		"type": "%s",
		"ttl": "%s",
		"redis": {
			"host": "%s",
			"port": %d,
			"database": %d
		}
	},
	"metrics": {
		"enabled": %v,
		"port": %d,
		"path": "%s"
	}
}`, config.Database.URL, config.Database.MaxConns, config.Database.EnableSSL, config.Database.Timeout,
			config.Cache.Type, config.Cache.TTL, config.Cache.Redis.Host, config.Cache.Redis.Port, config.Cache.Redis.Database,
			config.Metrics.Enabled, config.Metrics.Port, config.Metrics.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// Create server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      mux,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
		IdleTimeout:  config.Timeout * 2,
	}

	// Start metrics server if enabled
	if config.Metrics.Enabled {
		go startMetricsServer(config.Metrics)
	}

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("❌ Server shutdown error: %v", err)
		}
		log.Println("✅ Server shutdown complete")
	}()

	// Start server
	log.Printf("🚀 Server starting on %s:%d", config.Host, config.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("❌ Server error: %v", err)
	}
}

func startMetricsServer(metrics struct {
	Enabled bool   `flag:"metrics" env:"METRICS_ENABLED" description:"Enable metrics collection" default:"true"`
	Port    int    `flag:"metrics-port" env:"METRICS_PORT" description:"Metrics server port" default:"9090"`
	Path    string `flag:"metrics-path" env:"METRICS_PATH" description:"Metrics endpoint path" default:"/metrics"`
}) {
	mux := http.NewServeMux()
	mux.HandleFunc(metrics.Path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 42
http_requests_total{method="POST",status="201"} 15

# HELP memory_usage_bytes Current memory usage
# TYPE memory_usage_bytes gauge
memory_usage_bytes 12345678
`))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", metrics.Port),
		Handler: mux,
	}

	log.Printf("📊 Metrics server starting on port %d%s", metrics.Port, metrics.Path)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("❌ Metrics server error: %v", err)
	}
}

func printStartupInfo(config ServerConfig) {
	fmt.Println("🚀 Web Server Starting")
	fmt.Println("======================")
	fmt.Printf("Host: %s\n", config.Host)
	fmt.Printf("Port: %d\n", config.Port)
	fmt.Printf("Debug: %v\n", config.Debug)
	fmt.Printf("Log Level: %s\n", config.LogLevel)
	fmt.Printf("Workers: %d\n", config.Workers)
	fmt.Printf("Request Timeout: %v\n", config.Timeout)
	fmt.Printf("Allowed IPs: %v\n", config.AllowedIPs)
	fmt.Println()

	fmt.Println("📊 Database Configuration")
	fmt.Println("========================")
	fmt.Printf("URL: %s\n", config.Database.URL)
	fmt.Printf("Max Connections: %d\n", config.Database.MaxConns)
	fmt.Printf("SSL Enabled: %v\n", config.Database.EnableSSL)
	fmt.Printf("Timeout: %v\n", config.Database.Timeout)
	fmt.Printf("Auto-migrate: %v\n", config.Database.AutoMigrate)
	fmt.Println()

	fmt.Println("🗄️  Cache Configuration")
	fmt.Println("======================")
	fmt.Printf("Type: %s\n", config.Cache.Type)
	fmt.Printf("TTL: %s\n", config.Cache.TTL)
	if config.Cache.Type == "redis" {
		fmt.Printf("Redis Host: %s:%d\n", config.Cache.Redis.Host, config.Cache.Redis.Port)
		fmt.Printf("Redis Database: %d\n", config.Cache.Redis.Database)
	}
	fmt.Println()

	if config.Metrics.Enabled {
		fmt.Println("📈 Metrics Configuration")
		fmt.Println("=======================")
		fmt.Printf("Enabled: %v\n", config.Metrics.Enabled)
		fmt.Printf("Port: %d\n", config.Metrics.Port)
		fmt.Printf("Path: %s\n", config.Metrics.Path)
		fmt.Println()
	}

	fmt.Println("Available endpoints:")
	fmt.Printf("  http://%s:%d/         - Server status\n", config.Host, config.Port)
	fmt.Printf("  http://%s:%d/health   - Health check\n", config.Host, config.Port)
	fmt.Printf("  http://%s:%d/config   - Configuration details\n", config.Host, config.Port)
	if config.Metrics.Enabled {
		fmt.Printf("  http://localhost:%d%s  - Metrics\n", config.Metrics.Port, config.Metrics.Path)
	}
	fmt.Println()
}
