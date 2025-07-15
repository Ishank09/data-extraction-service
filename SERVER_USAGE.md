# Server Usage

## Starting the Server

```bash
go run cmd/main.go serve
```

The server will start on `http://localhost:8080`

## Main API Endpoints

### OAuth Flow
- `POST /api/v1/oauth/authorize` - Get authorization URL
- `GET /api/v1/documents` - Get documents (requires Authorization header)

### Health Check
- `GET /api/v1/health` - Health status
- `GET /ping` - Basic ping

## Environment Variables

The service uses environment variables from `.vscode/launch.json` configuration:

- `MSGRAPH_CLIENT_ID` - Microsoft Graph client ID
- `MSGRAPH_CLIENT_SECRET` - Microsoft Graph client secret  
- `MSGRAPH_TENANT_ID` - Tenant ID (use "common" for personal accounts)
- `OAUTH_REDIRECT_URI` - OAuth redirect URI
- `OAUTH_SCOPES` - OAuth scopes (comma-separated)

## Response Format

Documents API returns:

```json
{
  "source": "data_extraction_service",
  "documents": [
    {
      "id": "doc-1",
      "title": "Document Title",
      "content": "Document content...",
      "source": "msgraph",
      "type": "onenote_page"
    }
  ]
}
``` 