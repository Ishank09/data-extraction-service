# MongoDB Integration Guide

This guide explains how to use the MongoDB integration in the data-extraction-service. The service now automatically stores all processed documents in MongoDB and provides endpoints to retrieve and manage stored data.

## 🚀 Quick Start

### 1. Start MongoDB

```bash
# Using Docker
docker run -d --name mongodb -p 27017:27017 mongo:latest

# Or using MongoDB locally
mongod --dbpath /data/db
```

### 2. Configure Environment

```bash
# Basic MongoDB configuration
export MONGODB_URI="mongodb://localhost:27017"
export MONGODB_DATABASE="data_extraction_service"

# Optional: Authentication (if needed)
export MONGODB_USERNAME="your_username"
export MONGODB_PASSWORD="your_password"
export MONGODB_AUTH_SOURCE="admin"
```

### 3. Start the Service

```bash
./bin/data-extraction-service serve
```

## 📊 How It Works

### Automatic Document Storage

All pipeline endpoints now automatically store processed documents in MongoDB:

1. **Extract and Store**: When you call any pipeline endpoint, documents are processed AND stored
2. **Dual Response**: You get both the processed data AND storage confirmation
3. **Fault Tolerant**: If storage fails, you still get your processed data (with a warning header)

### Example Response with Storage Information

```json
{
  "source": "etl_pipeline",
  "fetched_at": "2024-01-01T00:00:00Z",
  "schema_version": "v1",
  "documents": [...],
  "document_count": 15,
  "storage": {
    "stored": true,
    "collection_id": "507f1f77bcf86cd799439011",
    "stored_documents": 15
  }
}
```

## 🔍 Retrieving Stored Documents

### Get All Documents

```bash
curl "http://localhost:8080/api/v1/documents"
```

### Filter by Source

```bash
curl "http://localhost:8080/api/v1/documents?source=static"
curl "http://localhost:8080/api/v1/documents?source=onenote"
```

### Filter by Type

```bash
curl "http://localhost:8080/api/v1/documents?type=pdf"
curl "http://localhost:8080/api/v1/documents?type=csv"
```

### Filter by Title (Regex Search)

```bash
curl "http://localhost:8080/api/v1/documents?title=report"
```

### Filter by Date Range

```bash
curl "http://localhost:8080/api/v1/documents?fetched_after=2024-01-01T00:00:00Z&fetched_before=2024-01-31T23:59:59Z"
```

### Pagination

```bash
curl "http://localhost:8080/api/v1/documents?limit=10&skip=20"
```

### Combined Filters

```bash
curl "http://localhost:8080/api/v1/documents?source=static&type=pdf&limit=5"
```

## 📈 Document Statistics

Get insights about your stored documents:

```bash
curl "http://localhost:8080/api/v1/documents/stats"
```

Response:
```json
{
  "total_documents": 150,
  "total_collections": 12,
  "documents_by_source": {
    "static": 45,
    "onenote": 105
  }
}
```

## 🗂️ Collection Metadata

View information about document collection runs:

```bash
curl "http://localhost:8080/api/v1/documents/collections"
```

## 🧹 Cleanup Old Documents

Remove documents older than a specified duration:

```bash
# Delete documents older than 30 days
curl -X DELETE "http://localhost:8080/api/v1/documents/cleanup?older_than=720h"

# Delete documents older than 7 days  
curl -X DELETE "http://localhost:8080/api/v1/documents/cleanup?older_than=168h"
```

## 🗄️ MongoDB Collections

The service creates two collections:

### 1. `documents` Collection

Stores individual documents with schema:

```json
{
  "_id": "ObjectId",
  "document_id": "unique_document_id",
  "source": "static|onenote",
  "type": "pdf|csv|txt|onenote",
  "title": "Document Title",
  "location": "file/path/or/url",
  "created_at": "2024-01-01T00:00:00Z",
  "fetched_at": "2024-01-01T00:00:00Z", 
  "stored_at": "2024-01-01T00:00:00Z",
  "content": "Extracted text content...",
  "metadata": {
    "file_type": "pdf",
    "word_count": 1250
  }
}
```

### 2. `document_collections` Collection

Stores metadata about each extraction run:

```json
{
  "_id": "ObjectId",
  "source": "etl_pipeline",
  "fetched_at": "2024-01-01T00:00:00Z",
  "stored_at": "2024-01-01T00:00:00Z",
  "schema_version": "v1",
  "document_count": 15,
  "document_ids": ["doc1", "doc2", "..."]
}
```

## 🔧 Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGODB_URI` | `mongodb://localhost:27017` | MongoDB connection string |
| `MONGODB_DATABASE` | `data_extraction_service` | Database name |
| `MONGODB_USERNAME` | - | Username for authentication |
| `MONGODB_PASSWORD` | - | Password for authentication |
| `MONGODB_AUTH_SOURCE` | `admin` | Authentication database |

### Advanced MongoDB URI Examples

```bash
# With authentication
export MONGODB_URI="mongodb://username:password@localhost:27017/database"

# MongoDB Atlas
export MONGODB_URI="mongodb+srv://username:password@cluster.mongodb.net/"

# Replica set
export MONGODB_URI="mongodb://host1:27017,host2:27017,host3:27017/database?replicaSet=myReplicaSet"
```

## 🚨 Troubleshooting

### MongoDB Connection Issues

1. **Check MongoDB is running**:
   ```bash
   mongosh --eval "db.runCommand({ping: 1})"
   ```

2. **Check service logs**:
   ```bash
   ./bin/data-extraction-service serve --verbose
   ```

3. **Verify connectivity**:
   ```bash
   curl "http://localhost:8080/api/v1/documents/health"
   ```

### Storage Warnings

If you see `X-Storage-Warning` headers in responses:

1. Check MongoDB connection
2. Verify database permissions
3. Check service logs for detailed error messages

### Performance Considerations

- **Indexing**: Consider adding indexes on frequently queried fields:
  ```javascript
  db.documents.createIndex({ "source": 1, "fetched_at": -1 })
  db.documents.createIndex({ "type": 1 })
  db.documents.createIndex({ "title": "text" })
  ```

- **Cleanup**: Regularly clean up old documents to maintain performance
- **Monitoring**: Use the stats endpoint to monitor storage growth

## 📝 Example Workflow

Here's a complete example of using the MongoDB integration:

```bash
# 1. Start MongoDB
docker run -d --name mongodb -p 27017:27017 mongo:latest

# 2. Start the service
export MONGODB_URI="mongodb://localhost:27017"
./bin/data-extraction-service serve

# 3. Extract and store documents
curl "http://localhost:8080/api/v1/pipeline"

# 4. View stored documents
curl "http://localhost:8080/api/v1/documents?limit=5"

# 5. Get statistics
curl "http://localhost:8080/api/v1/documents/stats"

# 6. Search for specific documents
curl "http://localhost:8080/api/v1/documents?type=pdf&title=report"

# 7. Clean up old data
curl -X DELETE "http://localhost:8080/api/v1/documents/cleanup?older_than=720h"
```

This integration makes your data-extraction-service a fully-fledged document processing and storage system! 