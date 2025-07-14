# Data Extraction Service - Server Usage

This document explains how to run the data extraction service server with Microsoft Graph integration.

## Environment Variables

The server supports the following environment variables:

### Required for basic operation:
- `PORT` - Server port (default: 8080)
- `ENVIRONMENT_NAME` - Environment name (set to "local" for local development)

### Optional for Microsoft Graph integration:
- `MSGRAPH_CLIENT_ID` - Microsoft Graph application client ID
- `MSGRAPH_CLIENT_SECRET` - Microsoft Graph application client secret
- `MSGRAPH_TENANT_ID` - Microsoft Graph tenant ID

**Note:** If MSGraph environment variables are not set, the server will work with static files only.

## Starting the Server

### Option 1: With Static Files Only
```bash
# No MSGraph environment variables needed
go run cmd/main.go serve
```

### Option 2: With Microsoft Graph Integration
```bash
# Set environment variables
export MSGRAPH_CLIENT_ID="your-client-id"
export MSGRAPH_CLIENT_SECRET="your-client-secret"
export MSGRAPH_TENANT_ID="your-tenant-id"

# Run the server
go run cmd/main.go serve
```

### Option 3: Using .env file (recommended)
```bash
# Create .env file
cat > .env << EOF
PORT=8080
ENVIRONMENT_NAME=local
MSGRAPH_CLIENT_ID=your-client-id
MSGRAPH_CLIENT_SECRET=your-client-secret
MSGRAPH_TENANT_ID=your-tenant-id
EOF

# Export variables and run
export $(cat .env | xargs) && go run cmd/main.go serve
```

## API Endpoints

Once the server is running, you can access the following endpoints:

### Data Extraction Endpoints

#### 1. Get All Documents
```bash
curl http://localhost:8080/api/v1/documents
```
Returns all documents from all available sources (static files + MSGraph if configured).

#### 2. Get Documents by Source
```bash
# Static files only
curl http://localhost:8080/api/v1/documents/static

# Microsoft Graph OneNote (if configured)
curl http://localhost:8080/api/v1/documents/msgraph
curl http://localhost:8080/api/v1/documents/onenote
```

#### 3. Get Documents by File Type
```bash
# Get JSON files
curl http://localhost:8080/api/v1/documents/type/json

# Get CSV files
curl http://localhost:8080/api/v1/documents/type/csv

# Supported types: json, csv, txt, pdf, html, xml
```

#### 4. Get Available Sources
```bash
curl http://localhost:8080/api/v1/sources
```
Returns information about available data sources and their status.

#### 5. Health Check
```bash
curl http://localhost:8080/api/v1/health
```
Returns health status of the handler and its components.

### Legacy Endpoints

#### System Health
```bash
curl http://localhost:8080/ping
```

#### Metrics
```bash
curl http://localhost:8080/metrics
```

## Example Response Format

All document endpoints return data in this format:

```json
{
  "source": "data_extraction_service",
  "fetched_at": "2024-01-01T00:00:00Z",
  "schema_version": "v1",
  "documents": [
    {
      "id": "doc-1",
      "source": "static",
      "type": "json",
      "title": "example.json",
      "location": "json/files/example.json",
      "created_at": "2024-01-01T00:00:00Z",
      "fetched_at": "2024-01-01T00:00:00Z",
      "content": "...",
      "metadata": {
        "filename": "example.json",
        "file_type": "json",
        "file_size": 1024
      }
    }
  ]
}
```

## Microsoft Graph Setup

To use Microsoft Graph integration, you need to:

1. **Register an application** in Azure Active Directory
2. **Grant permissions** for OneNote (Notes.Read, Notes.Read.All, User.Read)
3. **Create a client secret** for your application
4. **Set environment variables** with your application credentials

### Required Microsoft Graph Permissions:
- `Notes.Read` - Read user's OneNote notebooks
- `Notes.Read.All` - Read all OneNote notebooks  
- `User.Read` - Read user profile

## Troubleshooting

### MSGraph Not Working?
- Check that all three environment variables are set: `MSGRAPH_CLIENT_ID`, `MSGRAPH_CLIENT_SECRET`, `MSGRAPH_TENANT_ID`
- Verify your Azure AD application has the correct permissions
- Check the server logs for authentication errors

### No Documents Returned?
- For static files: Check if files exist in `pkg/static/*/files/` directories
- For MSGraph: Ensure the user has OneNote content and proper permissions

### Server Won't Start?
- Check if port 8080 is already in use
- Verify all imports are correctly installed: `go mod tidy`
- Check for any compilation errors: `go build ./cmd/...`

## Server Logs

The server provides detailed logging:
- **MSGraph configured**: "Creating data extraction handler with MSGraph integration"
- **Static only**: "Creating data extraction handler with static files only (MSGraph not configured)"
- **Port info**: "Running on port 8080"

## Development

### Running Tests
```bash
make t
```

### Building
```bash
go build ./cmd/...
```

### Adding New File Types
Add new processors to `pkg/static/` directory and update the `static.Client` to include them. 