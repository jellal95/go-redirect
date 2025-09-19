# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

This is a Go-based affiliate redirect service that intelligently routes traffic through bot filtering, geo-targeting, and weighted product distribution. The service acts as a middleman between ad networks (PropellerAds, Galaksion, Popcash, ClickAdilla) and affiliate links (Shopee, Lazada, etc.), with comprehensive logging and analytics.

## Common Development Commands

### Build and Run
```powershell
# Build the application
go build -o go-redirect.exe .

# Run locally
go run main.go

# Run with environment variables
$env:PORT="3000"; go run main.go
```

### Testing
```powershell
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test package
go test ./handlers/

# Run with coverage
go test -cover ./...
```

### Development Tools
```powershell
# Format code
go fmt ./...

# Vet for potential issues
go vet ./...

# Tidy modules
go mod tidy

# Download dependencies
go mod download
```

### Docker Operations
```powershell
# Build Docker image
docker build -t go-redirect .

# Run with Docker
docker run -p 8080:8080 -v ${PWD}/config:/app/config -v ${PWD}/logs:/logs go-redirect
```

### Deployment (Fly.io)
```powershell
# Deploy to Fly.io
fly deploy

# Check logs
fly logs

# Scale machines
fly scale count 2

# Connect to console
fly console
```

## High-Level Architecture

### Core Components

**Main Application Flow:**
1. **Bot Filter Middleware** (`middleware/bot_middleware.go`) - Primary traffic filter that validates requests based on geo-location (Indonesia-only), device type (mobile-only), user agents, IP ranges, and rate limiting
2. **Redirect Handler** (`handlers/redirect.go`) - Core business logic that selects products using weighted randomization and builds affiliate URLs with proper parameter mapping
3. **Configuration System** (`config/config.yaml`, `utils/config.go`) - YAML-based configuration for ad networks and product definitions with percentage-based weighting
4. **Logging Infrastructure** (`utils/logger.go`) - Comprehensive request logging to both memory and daily JSONL files with structured analytics

### Data Flow Architecture

**Request Processing Pipeline:**
```
Incoming Request → Bot Filter → Route Handler → Product Selection → URL Building → Redirect
                     ↓
               Geo Validation (Indonesia only)
                     ↓
               Device Check (Mobile only) 
                     ↓
               Rate Limiting & IP Filtering
```

**Product Selection Logic:**
- Products are selected using weighted randomization based on `percentage` field
- Fallback to CSV configuration if YAML products not available
- Supports both direct product selection (`?product=ID`) and random weighted selection

**Parameter Mapping System:**
- Dynamic placeholder replacement: `{click_id}`, `{campaign_id}`, `{spot_id}`, etc.
- Ad network specific parameter extraction (PropellerAds uses `subid`, Galaksion/Popcash use `clickid`)
- Extra query parameters automatically appended to final URLs

### Key Integrations

**Ad Networks:**
- PropellerAds (type_ads=1): Uses `aid`, `tid`, `visitor_id` for postbacks
- Galaksion (type_ads=2): Uses `cid`, `click_id` for postbacks  
- Popcash (type_ads=3): Uses `aid`, `type`, `clickid` for postbacks
- ClickAdilla (type_ads=4): Uses `token`, `campaign_id`, `click_id` for postbacks

**Geo-targeting:**
- MaxMind GeoIP2 database integration for country/city detection
- Configurable country allowlists (currently ID/Indonesia only)
- Comprehensive geo analytics in logs

**Bot Detection:**
- User-Agent blacklisting (curl, bot, spider, crawler, python, scrapy, headless)
- IP prefix blocking (Google Cloud, AWS, Cloudflare, local datacenters)  
- Referrer domain filtering with regex support
- Rate limiting (10 requests per 10 seconds per IP)

## Development Guidelines

### Configuration Management
- Main config: `config/config.yaml` for ad networks and product definitions
- Product weighting: Use `percentage` field for traffic distribution
- CSV fallback: `config/config.csv` for alternative product loading
- GeoIP databases: Place `.mmdb` files in root directory

### Testing Approach
- Unit tests in `handlers/redirect_handler_test.go` cover URL building logic
- Test different parameter combinations and edge cases
- Use httptest for handler testing
- Mock external dependencies (GeoIP, file system)

### Logging and Monitoring  
- Structured logs in JSONL format stored in `logs/` directory
- Log analytics available at `/logs` endpoint with comprehensive summaries
- Postback tracking available at `/postbacks` endpoint
- Use LogInfo(), LogFatal() functions for consistent logging

### Performance Considerations
- In-memory request rate limiting with garbage collection
- Concurrent-safe logging with mutex protection
- Efficient regex compilation for referrer filtering  
- Percentage-based product selection with O(n) complexity

### Security Notes
- Bypass mechanism with hardcoded key (`a9f7x2kq`) for debugging
- IP-based rate limiting to prevent abuse
- Comprehensive request logging for security monitoring
- Geo-restriction to Indonesia traffic only

## File Structure Overview

- `main.go` - Application entry point and Fiber setup
- `handlers/` - HTTP request handlers (redirect, postback, logs, etc.)
- `middleware/` - Bot filtering and request validation
- `models/` - Data structures and constants
- `utils/` - Helper functions (logging, config, URL building)
- `geo/` - Geolocation functionality
- `config/` - Configuration files (YAML and CSV)
- `views/` - HTML templates for pre-sale pages
- `logs/` - Runtime log storage (mounted volume in production)

## Common Debugging Scenarios

### Bot Filter Issues
- Check `/logs` endpoint for `block_request` entries
- Use bypass parameter: `?bypass=a9f7x2kq` for testing
- Verify GeoIP database presence and country codes

### Product Selection Problems
- Validate YAML configuration syntax in `config/config.yaml`
- Check percentage sum and product availability
- Test with specific product: `?product=ID`

### URL Building Errors
- Review placeholder mapping in `utils/helpers.go`
- Check parameter extraction logic in redirect handler
- Verify affiliate URL templates have correct placeholder syntax

### Postback Failures
- Monitor `/postbacks` endpoint for received callbacks
- Check ad network configuration in config file
- Verify parameter mapping matches network requirements