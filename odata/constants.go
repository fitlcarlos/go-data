package odata

import "time"

// Validation Limits
const (
	DefaultMaxFilterLength  = 5000   // 5KB max filter string
	DefaultMaxSearchLength  = 1000   // 1KB max search string
	DefaultMaxSelectLength  = 1000   // 1KB max select string
	DefaultMaxOrderByLength = 500    // 500 bytes max orderby string
	DefaultMaxExpandDepth   = 5      // Max 5 levels of expand nesting
	DefaultMaxTopValue      = 1000   // Max 1000 records per page
	DefaultMaxSkipValue     = 100000 // Max skip value
)

// Rate Limiting
const (
	DefaultRateLimitPerMinute = 100         // 100 requests per minute
	DefaultRateLimitBurstSize = 10          // Allow 10 concurrent requests
	DefaultRateLimitWindow    = time.Minute // 1 minute window
)

// Connection Pool
const (
	DefaultMinConnections  = 2                // Min connections per pool
	DefaultMaxConnections  = 10               // Max connections per pool
	DefaultMaxIdleTime     = 10 * time.Minute // Max idle time
	DefaultConnMaxLifetime = time.Hour        // Max connection lifetime
)

// Query Defaults
const (
	DefaultPageSize = 50               // Default page size if $top not specified
	DefaultTimeout  = 30 * time.Second // Default query timeout
)

// Security
const (
	MinJWTSecretLength       = 32                 // Minimum JWT secret length
	DefaultTokenExpiration   = 15 * time.Minute   // Default access token expiration
	DefaultRefreshExpiration = 7 * 24 * time.Hour // 7 days refresh token
)

// Health Check
const (
	DefaultHealthCheckPeriod = 30 * time.Second // Health check interval
	MaxFailureCount          = 3                // Mark unhealthy after 3 failures
)

// Logging
const (
	MaxLogEntrySize = 10000 // Max size of single log entry
)

// Property Name Validation
const (
	MaxPropertyNameLength = 100                 // Max property name length
	PropertyNamePattern   = `^[a-zA-Z0-9_\.]+$` // Allowed chars in property names
)
