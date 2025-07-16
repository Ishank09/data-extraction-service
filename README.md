# Data Extraction Service

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.24.1-blue)](https://golang.org/)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/ishank09/data-extraction-service)

A powerful, open-source **ETL (Extract, Transform, Load) service** built in Go that converts documents from multiple data sources into unified JSON format. Extract content from static files and Microsoft Graph OneNote notebooks, transform them into consistent JSON structure, and serve through a modern REST API.

## 🌟 Features

- **🔄 Universal File Converter**: Transform CSV, PDF, TXT, XML, HTML, JSON, and OneNote files into structured JSON
- **📁 Embedded File Processing**: Works out-of-the-box with embedded static files 
- **🔧 Extensible Architecture**: Easy to add new file types and data sources
- **☁️ Microsoft Graph Integration**: Optional OneNote and Office 365 document access
- **🔒 OAuth 2.0 Authentication**: Secure Microsoft Graph integration
- **🌐 RESTful API**: Clean, documented endpoints for easy integration
- **🎯 Zero Configuration**: Works without any setup for static files
- **🔀 Flexible Deployment**: Static-only mode or full Microsoft Graph integration
- **🏢 Multi-tenant Support**: Personal and organizational Microsoft accounts
- **⚡ High Performance**: Concurrent OneNote processing with configurable worker pools

## 🚀 Quick Start

### Option 1: Static Files Only (No Setup Required)

```bash
# Clone the repository
git clone https://github.com/ishank09/data-extraction-service.git
cd data-extraction-service

# Install dependencies
go mod tidy

# Start the service
go run cmd/main.go serve
```

Visit `http://localhost:8080/api/v1/pipeline` to see your embedded files converted to JSON!

### Option 2: With Microsoft Graph Integration

1. **Set up Azure App Registration** (see [detailed instructions](#-azure-app-registration))
2. **Configure environment variables**:
```bash
export MSGRAPH_CLIENT_ID="your-client-id"
export MSGRAPH_CLIENT_SECRET="your-client-secret"
export MSGRAPH_TENANT_ID="common"
export OAUTH_REDIRECT_URI="http://localhost:8080/api/v1/oauth/callback"
export OAUTH_SCOPES="User.Read,Files.Read,Notes.Read,offline_access"
```
3. **Start the service**:
```bash
go run cmd/main.go serve
```

## 📁 Adding Your Own Files

The service uses Go's embed functionality to include files at compile time. Add your files to the appropriate directories:

### 📂 File Locations

```
pkg/static/
├── csv/files/          # Add your .csv files here
├── json/files/         # Add your .json files here  
├── txt/files/          # Add your .txt files here
├── pdf/files/          # Add your .pdf files here
├── html/files/         # Add your .html files here
└── xml/files/          # Add your .xml files here
```

### 📝 Example: Adding Files

```bash
# Add your CSV data
cp my-data.csv pkg/static/csv/files/

# Add your PDFs
cp report.pdf pkg/static/pdf/files/

# Add your text files  
cp notes.txt pkg/static/txt/files/

# Rebuild and restart
go run cmd/main.go serve
```

**Important**: After adding files, restart the service as files are embedded at compile time.

## 🏗️ ETL Architecture

### Extract Phase
- **Static Files**: Embedded files using Go's `//go:embed` directive
- **Microsoft Graph**: OneNote notebooks and pages via Graph API
- **File Types**: CSV, JSON, TXT, PDF, XML, HTML, OneNote

### Transform Phase
- **Content Parsing**: 
  - **CSV** → Structured rows and columns
  - **PDF** → Extracted text using go-fitz
  - **HTML** → Clean text extraction
  - **XML** → Parsed structure
  - **OneNote** → Rich content with metadata
- **Schema Normalization**: Unified document structure
- **JSON Serialization**: Consistent output format

### Load Phase
- **REST API**: JSON documents via HTTP endpoints
- **Real-time Processing**: On-demand transformation
- **Structured Metadata**: Rich document information

## 🔌 API Reference

### Core Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/api/v1/pipeline` | GET | Extract all data from available sources and store in MongoDB | Optional |
| `/api/v1/pipeline/static` | GET | Extract static files only (CSV, PDF, etc.) and store | No |
| `/api/v1/pipeline/msgraph` | GET | Extract OneNote data only and store | Yes |
| `/api/v1/pipeline/type/{type}` | GET | Extract data filtered by file type and store | No |
| `/api/v1/sources` | GET | Available data sources | No |

### Document Storage Endpoints (MongoDB)

| Endpoint | Method | Description | Query Parameters |
|----------|--------|-------------|------------------|
| `/api/v1/documents` | GET | Retrieve stored documents | `source`, `type`, `title`, `fetched_after`, `fetched_before`, `limit`, `skip` |
| `/api/v1/documents/collections` | GET | Retrieve document collection metadata | `source`, `fetched_after`, `fetched_before`, `limit`, `skip` |
| `/api/v1/documents/stats` | GET | Get document storage statistics | None |
| `/api/v1/documents/cleanup` | DELETE | Delete old documents | `older_than` (duration, e.g., "720h") |
| `/api/v1/documents/health` | GET | Document storage service health | None |

### Authentication Endpoints (OAuth)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/oauth/authorize` | POST | Get authorization URL |
| `/api/v1/oauth/callback` | GET | OAuth callback |
| `/api/v1/oauth/refresh` | POST | Refresh access token |
| `/api/v1/oauth/test` | POST | Validate token |

### Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/health` | GET | Service health status |
| `/ping` | GET | Basic ping |
| `/metrics` | GET | Prometheus metrics |

## 📊 JSON Output Format

All documents are transformed into a unified schema and automatically stored in MongoDB:

```json
{
  "source": "etl_pipeline",
  "fetched_at": "2024-01-01T00:00:00Z",
  "schema_version": "v1",
  "documents": [
    {
      "id": "pdf_report_1234567890",
      "title": "quarterly-report.pdf",
      "content": "Extracted text content from PDF...",
      "source": "embedded",
      "type": "pdf",
      "location": "files/quarterly-report.pdf",
      "created_at": "2024-01-01T00:00:00Z",
      "fetched_at": "2024-01-01T00:00:00Z",
      "metadata": {
        "filename": "quarterly-report.pdf",
        "file_type": "pdf",
        "word_count": 1250,
        "page_count": 5
      }
    }
  ],
  "document_count": 1,
  "storage": {
    "stored": true,
    "collection_id": "507f1f77bcf86cd799439011",
    "stored_documents": 1
  }
}
```

## ⚙️ Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | 8080 | Server port |
| `ENVIRONMENT_NAME` | No | local | Environment name |

#### Microsoft Graph Configuration
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MSGRAPH_CLIENT_ID` | No | - | Azure AD application client ID |
| `MSGRAPH_CLIENT_SECRET` | No | - | Azure AD application client secret |
| `MSGRAPH_TENANT_ID` | No | - | Azure AD tenant ID or "common" |
| `MSGRAPH_USER_ID` | No | - | User ID for application flow |

#### OAuth Configuration  
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OAUTH_REDIRECT_URI` | No | - | OAuth redirect URI |
| `OAUTH_SCOPES` | No | - | Comma-separated OAuth scopes |

#### MongoDB Configuration
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MONGODB_URI` | **Yes** | *None* | MongoDB connection URI |
| `MONGODB_DATABASE` | **Yes** | *None* | Database name |
| `MONGODB_USERNAME` | No | - | MongoDB username |
| `MONGODB_PASSWORD` | No | - | MongoDB password |
| `MONGODB_AUTH_SOURCE` | No | admin | Authentication database |

> ⚠️ **Note**: MongoDB integration is optional. If `MONGODB_URI` is not provided, the service will run without document storage.

#### Performance Tuning
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ONENOTE_SECTION_WORKERS` | No | `5` | Max concurrent section workers |
| `ONENOTE_CONTENT_WORKERS` | No | `10` | Max concurrent content workers |

### 🔐 Azure App Registration

#### 1. Create App Registration
1. Go to [Azure Portal](https://portal.azure.com) → **Azure Active Directory** → **App registrations**
2. Click **New registration**
3. Configure:
   - **Name**: `data-extraction-service`
   - **Account types**: Choose based on your needs:
     - Personal accounts: "Personal Microsoft accounts only"
     - Work/School: "Accounts in this organizational directory only"
     - Both: "Accounts in any organizational directory and personal Microsoft accounts"
   - **Redirect URI**: `http://localhost:8080/api/v1/oauth/callback`

#### 2. Get Credentials
- **Client ID**: Copy from Overview → Application (client) ID
- **Client Secret**: Certificates & secrets → New client secret → Copy **Value**
- **Tenant ID**: Copy from Overview → Directory (tenant) ID

#### 3. Configure Permissions
Add these Microsoft Graph permissions:
- `User.Read` - Read user profile
- `Files.Read` - Read user files  
- `Notes.Read` - Read OneNote notebooks
- `offline_access` - Refresh tokens

## 🛠️ Development

### Build Commands

```bash
# Build binary
make build

# Run tests
make test

# Test with coverage
make test-coverage

# Run linter
make lint

# Clean artifacts
make clean

# Install dependencies
make deps

# Generate mocks
make mocks
```

### Development Server

```bash
# Development mode with auto-reload
make dev

# With verbose logging
go run cmd/main.go serve --verbose
```

## 🔧 Usage Patterns

### 1. Static Files Only
Perfect for processing embedded documents without external dependencies:

```bash
# No environment variables needed
go run cmd/main.go serve

# Extract all data
curl http://localhost:8080/api/v1/pipeline

# Extract specific file types
curl http://localhost:8080/api/v1/pipeline/type/pdf
```

### 2. Microsoft Graph Integration
Process OneNote documents alongside static files:

```bash
# Set MSGraph environment variables
export MSGRAPH_CLIENT_ID="your-client-id"
export MSGRAPH_CLIENT_SECRET="your-client-secret"
export MSGRAPH_TENANT_ID="common"

# Start service
go run cmd/main.go serve

# Get authorization URL
curl -X POST http://localhost:8080/api/v1/oauth/authorize

# After OAuth flow, extract data with token
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/v1/pipeline
```

### 3. Hybrid Mode
Best of both worlds - static files work immediately, MSGraph when configured:

```bash
# Works with or without MSGraph configuration
go run cmd/main.go serve

# Returns static files immediately
# Returns OneNote data if MSGraph is configured
curl http://localhost:8080/api/v1/pipeline
```

## 📖 API Examples

### Extract All Data
```bash
curl http://localhost:8080/api/v1/pipeline
```

### Extract Static Files Only
```bash
curl http://localhost:8080/api/v1/pipeline/static
```

### Extract OneNote Data (with authentication)
```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     http://localhost:8080/api/v1/pipeline/msgraph
```

### Filter by File Type
```bash
# Extract only PDF data
curl http://localhost:8080/api/v1/pipeline/type/pdf

# Extract only CSV data  
curl http://localhost:8080/api/v1/pipeline/type/csv
```

### Check Available Sources
```bash
curl http://localhost:8080/api/v1/sources
```

## 🔍 Supported File Types

| Type | Extensions | Processing | Output |
|------|------------|------------|---------|
| **CSV** | `.csv` | Parse rows/columns | Structured JSON data |
| **PDF** | `.pdf` | Text extraction (go-fitz) | Plain text content |
| **TXT** | `.txt` | Direct content | Raw text |
| **HTML** | `.html`, `.htm` | Clean text extraction | Stripped content |
| **XML** | `.xml` | Structure parsing | Parsed elements |
| **JSON** | `.json` | Validation & normalization | Structured data |
| **OneNote** | N/A | Rich content extraction | Formatted content |

## 🚨 Troubleshooting

### Common Issues

#### "No documents found"
- **Cause**: No files in static directories
- **Solution**: Add files to `pkg/static/*/files/` and restart

#### "MSGraph handler not configured"  
- **Cause**: Missing environment variables
- **Solution**: Either set MSGraph variables or use static-only mode

#### "Access token invalid"
- **Cause**: Expired or invalid OAuth token
- **Solution**: Refresh token via `/api/v1/oauth/refresh`

#### "PDF extraction failed"
- **Cause**: Corrupted or password-protected PDF
- **Solution**: Document will still appear with metadata

### Debug Mode
```bash
# Enable verbose logging
ENVIRONMENT_NAME=local go run cmd/main.go serve --verbose

# Check health status
curl http://localhost:8080/api/v1/health

# Verify available sources
curl http://localhost:8080/api/v1/sources
```

## 🤝 Contributing

We welcome contributions! Here's how to get started:

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Add** your files to appropriate directories
4. **Test** your changes: `make test`
5. **Commit** your changes: `git commit -m 'Add amazing feature'`
6. **Push** to the branch: `git push origin feature/amazing-feature`
7. **Open** a Pull Request

### Development Setup
```bash
# Install development tools
make install-lint
make install-mockery

# Run full test suite
make check

# Generate test coverage
make test-coverage
```

## 📦 Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o data-extraction-service cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/data-extraction-service .
CMD ["./data-extraction-service", "serve"]
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🌟 Why This Project?

- **🎯 Simplicity**: Works out-of-the-box with embedded files
- **🔧 Flexibility**: Optional cloud integration 
- **🚀 Performance**: Efficient Go-based processing
- **📖 Extensibility**: Easy to add new file types
- **🔓 Open Source**: MIT licensed, community-driven
- **🏢 Production Ready**: Monitoring, logging, testing included

## ⚡ Performance Optimizations

### Concurrent OneNote Processing

The service uses **concurrent processing** for OneNote data extraction to significantly improve performance:

#### Performance Improvements
- **5-10x faster** OneNote data extraction compared to sequential processing
- **Worker pools** to limit concurrent API calls and respect rate limits
- **Parallel section processing**: Multiple sections fetched simultaneously
- **Parallel content fetching**: Multiple page contents fetched concurrently
- **Graceful error handling**: Individual failures don't stop the entire process

#### How It Works
1. **Section Workers**: Process multiple notebook sections in parallel
2. **Content Workers**: Fetch page content from multiple pages simultaneously
3. **Channels & Goroutines**: Efficient work distribution using Go's native concurrency
4. **Rate Limiting**: Configurable worker limits to avoid overwhelming the API

#### Tuning Performance

Configure concurrent workers via environment variables:

```bash
# Conservative settings (good for rate-limited APIs)
export ONENOTE_SECTION_WORKERS=3
export ONENOTE_CONTENT_WORKERS=5

# Default settings (balanced)
export ONENOTE_SECTION_WORKERS=5
export ONENOTE_CONTENT_WORKERS=10

# Aggressive settings (if your API limits allow)
export ONENOTE_SECTION_WORKERS=8
export ONENOTE_CONTENT_WORKERS=15
```

#### Performance Monitoring

The service logs performance metrics:
```
⚡ Performance: Used 5 section workers, 10 content workers
📊 Concurrent page fetching completed: 45 total pages found
📊 Concurrent content fetching completed: 43/45 pages successful
```

#### When to Tune

- **Increase workers** if you have high API rate limits
- **Decrease workers** if you encounter rate limiting errors
- **Monitor logs** for optimal worker counts for your use case

## 📚 Resources

- [Go Documentation](https://golang.org/doc/)
- [Microsoft Graph API](https://docs.microsoft.com/en-us/graph/)
- [Azure App Registration Guide](https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app)
- [OAuth 2.0 Authorization Code Flow](https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-auth-code-flow)

---

**Made with ❤️ by [Ishank Vasania](https://github.com/ishank09)**

*Transform your documents into structured JSON data effortlessly!*

