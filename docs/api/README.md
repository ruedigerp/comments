# Comment API Documentation

## 🌐 Base URL

```
https://comments.example.com
```

## 📋 Table of Contents

1. [Public Endpoints](#public-endpoints)
2. [Protected Admin Endpoints](#protected-admin-endpoints)
3. [Static Files](#static-files)
4. [Health & Monitoring](#health--monitoring)
5. [Authentication](#authentication)
6. [Error Responses](#error-responses)

-----

## 🔓 Public Endpoints

### 1. Create Comment

Create a new comment for a blog post.

```bash
POST /api/comments
```

**Headers:**

```
Content-Type: application/json
```

**Request Body:**

```json
{
  "post_id": "string",     // Required: Blog post identifier
  "username": "string",    // Required: Commenter's name
  "mailaddress": "string", // Required: Commenter's email
  "text": "string"         // Required: Comment text
}
```

**Example:**

```bash
curl -X POST "https://comments.example.com/api/comments" \
  -H "Content-Type: application/json" \
  -d '{
    "post_id": "2025-06-19-git-merge-script",
    "username": "John Doe",
    "mailaddress": "john@example.com",
    "text": "Great article! Thanks for sharing."
  }'
```

**Response (201 Created):**

```json
{
  "id": 42,
  "post_id": "2025-06-19-git-merge-script",
  "username": "John Doe",
  "mailaddress": "john@example.com",
  "text": "Great article! Thanks for sharing.",
  "active": true,
  "created_at": "2025-06-21T10:30:00Z"
}
```

-----

### 2. Get Comments

Retrieve comments for a specific post or all comments.

```bash
GET /api/comments
```

**Query Parameters:**

- `post_id` (optional): Filter by specific blog post
- `include_inactive` (optional): Include inactive comments (default: false)

**Examples:**

**Get all active comments:**

```bash
curl "https://comments.example.com/api/comments"
```

**Get comments for specific post:**

```bash
curl "https://comments.example.com/api/comments?post_id=2025-06-19-git-merge-script"
```

**Get all comments including inactive:**

```bash
curl "https://comments.example.com/api/comments?include_inactive=true"
```

**Response (200 OK):**

```json
[
  {
    "id": 42,
    "post_id": "2025-06-19-git-merge-script",
    "username": "John Doe",
    "mailaddress": "john@example.com",
    "text": "Great article! Thanks for sharing.",
    "active": true,
    "created_at": "2025-06-21T10:30:00Z"
  }
]
```

-----

### 3. Get Single Comment

Retrieve a specific comment by ID.

```bash
GET /api/comments/{id}
```

**Path Parameters:**

- `id`: Comment ID (integer)

**Example:**

```bash
curl "https://comments.example.com/api/comments/42"
```

**Response (200 OK):**

```json
{
  "id": 42,
  "post_id": "2025-06-19-git-merge-script",
  "username": "John Doe",
  "mailaddress": "john@example.com",
  "text": "Great article! Thanks for sharing.",
  "active": true,
  "created_at": "2025-06-21T10:30:00Z"
}
```

-----

## 🔐 Protected Admin Endpoints

All admin endpoints require authentication. See [Authentication](#authentication) section.

### 1. Update Comment Status

Activate or deactivate a comment.

```bash
PUT /api/comments/{id}/status
```

**Headers:**

```
Authorization: Bearer {admin_token}
Content-Type: application/json
```

**Path Parameters:**

- `id`: Comment ID (integer)

**Request Body:**

```json
{
  "active": true  // true to activate, false to deactivate
}
```

**Example:**

```bash
# Activate comment
curl -X PUT "https://comments.example.com/api/comments/42/status" \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"active": true}'

# Deactivate comment
curl -X PUT "https://comments.example.com/api/comments/42/status" \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"active": false}'
```

**Response (200 OK):**

```json
{
  "message": "Status aktualisiert"
}
```

-----

### 2. Delete Comment

Permanently delete a comment.

```bash
DELETE /api/comments/{id}
```

**Headers:**

```
Authorization: Bearer {admin_token}
```

**Path Parameters:**

- `id`: Comment ID (integer)

**Example:**

```bash
curl -X DELETE "https://comments.example.com/api/comments/42" \
  -H "Authorization: Bearer your-admin-token"
```

**Response (200 OK):**

```json
{
  "message": "Kommentar gelöscht"
}
```

-----

### 3. Admin Statistics

Get comprehensive statistics about comments.

```bash
GET /api/comments/admin/info
```

**Headers:**

```
Authorization: Bearer {admin_token}
```

**Example:**

```bash
curl "https://comments.example.com/api/comments/admin/info" \
  -H "Authorization: Bearer your-admin-token"
```

**Response (200 OK):**

```json
{
  "total_comments": 150,
  "active_comments": 142,
  "inactive_comments": 8,
  "unique_posts": 25,
  "recent_comments": 12,
  "top_posts": [
    {
      "post_id": "2025-06-19-git-merge-script",
      "comment_count": 15
    }
  ],
  "server_time": "2025-06-21T10:30:00Z",
  "version": "1.0.0",
  "stage": "production"
}
```

-----

## 📁 Static Files

### 1. Comment Widget JavaScript

Dynamic JavaScript widget for embedding comments.

```bash
GET /js/comment-widget.js
```

**Headers:**

```
Content-Type: application/javascript; charset=utf-8
Cache-Control: public, max-age=1800
Access-Control-Allow-Origin: *
```

**Example:**

```bash
curl "https://comments.example.com/js/comment-widget.js"
```

**Usage in HTML:**

```html
<script src="https://comments.example.com/js/comment-widget.js"></script>
<div data-comment-post-id="your-post-id"></div>
```

-----

### 2. CSS Files

Optional CSS files for custom styling.

```bash
GET /css/{filename}.css
```

**Example:**

```bash
curl "https://comments.example.com/css/comment-widget.css"
```

-----

### 3. General Static Files

Serve static assets like images, fonts, etc.

```bash
GET /static/{path}
```

**Example:**

```bash
curl "https://comments.example.com/static/images/logo.png"
```

-----

## 💚 Health & Monitoring

### 1. Health Check

Basic health check with system information.

```bash
GET /health
```

**Example:**

```bash
curl "https://comments.example.com/health"
```

**Response (200 OK):**

```json
{
  "status": "healthy",
  "service": "comment-api",
  "version": "1.0.0",
  "stage": "production",
  "timestamp": "2025-06-21T10:30:00Z",
  "uptime": "2h15m30s",
  "dependencies": {
    "redis": {
      "status": "healthy",
      "error": ""
    }
  },
  "system": {
    "go_version": "go1.23.0",
    "num_goroutines": 8,
    "memory_mb": 12,
    "gc_cycles": 3,
    "platform": "linux/amd64"
  },
  "endpoints": {
    "health": "/health",
    "api": "/api/comments",
    "admin": "/admin",
    "widget": "/js/comment-widget.js"
  }
}
```

-----

### 2. Liveness Check (Kubernetes)

Simple liveness probe for Kubernetes.

```bash
GET /health/live
```

**Example:**

```bash
curl "https://comments.example.com/health/live"
```

**Response (200 OK):**

```json
{
  "status": "alive",
  "time": "2025-06-21T10:30:00Z"
}
```

-----

### 3. Readiness Check (Kubernetes)

Readiness probe that checks Redis connectivity.

```bash
GET /health/ready
```

**Example:**

```bash
curl "https://comments.example.com/health/ready"
```

**Response (200 OK if ready, 503 if not):**

```json
{
  "status": "ready"
}
```

-----

### 4. Metrics

Application metrics for monitoring.

```bash
GET /metrics
```

**Example:**

```bash
curl "https://comments.example.com/metrics"
```

**Response (200 OK):**

```json
{
  "metrics": {
    "comments_total": 150,
    "comments_active": 142,
    "comments_inactive": 8,
    "posts_with_comments": 25,
    "uptime_seconds": 8130.5,
    "memory_bytes": 12582912,
    "goroutines": 8,
    "gc_cycles": 3
  },
  "timestamp": "2025-06-21T10:30:00Z"
}
```

-----

### 5. Simple Health Check

Minimal health check endpoint.

```bash
GET /
```

**Example:**

```bash
curl "https://comments.example.com/"
```

-----

## 🎛️ Admin Panel

### Admin Web Interface

Web-based admin panel for managing comments.

```bash
GET /admin
GET /admin/
```

**Query Parameters:**

- `token` (optional): Pre-fill admin token for auto-login

**Example:**

```bash
# Open in browser
https://comments.example.com/admin

# With auto-login
https://comments.example.com/admin?token=your-admin-token
```

-----

## 🔐 Authentication

### Admin Token Authentication

Admin endpoints require authentication using one of these methods:

#### 1. Authorization Bearer Header (Recommended)

```bash
curl -H "Authorization: Bearer your-admin-token" \
  "https://comments.example.com/api/comments/admin/info"
```

#### 2. Custom X-Admin-Token Header

```bash
curl -H "X-Admin-Token: your-admin-token" \
  "https://comments.example.com/api/comments/admin/info"
```

#### 3. Query Parameter (Less Secure)

```bash
curl "https://comments.example.com/api/comments/admin/info?token=your-admin-token"
```

### Token Management

- Tokens are configured via `ADMIN_TOKEN` environment variable
- Tokens should be at least 32 characters long
- Use `openssl rand -hex 32` to generate secure tokens

-----

## ❌ Error Responses

### HTTP Status Codes

- `200` - Success
- `201` - Created (for new comments)
- `400` - Bad Request (missing required fields)
- `401` - Unauthorized (invalid/missing admin token)
- `404` - Not Found (comment/endpoint doesn’t exist)
- `405` - Method Not Allowed (wrong HTTP method)
- `500` - Internal Server Error

### Error Response Format

```json
{
  "error": "Error description",
  "timestamp": "2025-06-21T10:30:00Z"
}
```

### Common Errors

#### 400 Bad Request

```bash
curl -X POST "https://comments.example.com/api/comments" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response:**

```
HTTP/1.1 400 Bad Request
Alle Felder sind erforderlich
```

#### 401 Unauthorized

```bash
curl "https://comments.example.com/api/comments/admin/info"
```

**Response:**

```json
{
  "error": "Missing authentication token",
  "timestamp": "2025-06-21T10:30:00Z"
}
```

#### 404 Not Found

```bash
curl "https://comments.example.com/api/comments/99999"
```

**Response:**

```
HTTP/1.1 404 Not Found
Kommentar nicht gefunden
```

-----

## 🧪 Complete Testing Examples

### Test Comment Workflow

```bash
#!/bin/bash
API_BASE="https://comments.example.com"
ADMIN_TOKEN="your-admin-token"

echo "🧪 Testing Comment API Workflow"
echo "================================"

# 1. Health Check
echo "1. Health Check..."
curl -s "$API_BASE/health" | jq .status

# 2. Create Comment
echo "2. Creating comment..."
COMMENT_ID=$(curl -s -X POST "$API_BASE/api/comments" \
  -H "Content-Type: application/json" \
  -d '{
    "post_id": "test-post",
    "username": "Test User",
    "mailaddress": "test@example.com",
    "text": "This is a test comment"
  }' | jq -r .id)

echo "Created comment with ID: $COMMENT_ID"

# 3. Get Comment
echo "3. Retrieving comment..."
curl -s "$API_BASE/api/comments/$COMMENT_ID" | jq .

# 4. Get Comments for Post
echo "4. Getting comments for post..."
curl -s "$API_BASE/api/comments?post_id=test-post" | jq length

# 5. Admin: Get Statistics
echo "5. Getting admin statistics..."
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/api/comments/admin/info" | jq .total_comments

# 6. Admin: Deactivate Comment
echo "6. Deactivating comment..."
curl -s -X PUT -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"active": false}' \
  "$API_BASE/api/comments/$COMMENT_ID/status" | jq .message

# 7. Verify Comment is Hidden
echo "7. Verifying comment is hidden..."
curl -s "$API_BASE/api/comments?post_id=test-post" | jq length

# 8. Admin: Delete Comment
echo "8. Deleting comment..."
curl -s -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/api/comments/$COMMENT_ID" | jq .message

echo "✅ Test workflow completed!"
```

### Performance Testing

```bash
# Load test - create multiple comments
for i in {1..10}; do
  curl -s -X POST "https://comments.example.com/api/comments" \
    -H "Content-Type: application/json" \
    -d "{
      \"post_id\": \"load-test\",
      \"username\": \"User$i\",
      \"mailaddress\": \"user$i@example.com\",
      \"text\": \"Load test comment $i\"
    }" &
done
wait

# Check results
curl -s "https://comments.example.com/api/comments?post_id=load-test" | jq length
```

-----

## 📋 Quick Reference

### Essential Endpoints

```bash
# Public
POST   /api/comments              # Create comment
GET    /api/comments              # Get comments
GET    /api/comments/{id}         # Get single comment

# Admin (requires auth)
PUT    /api/comments/{id}/status  # Toggle active status
DELETE /api/comments/{id}         # Delete comment
GET    /api/comments/admin/info   # Statistics

# Static
GET    /js/comment-widget.js      # Widget JavaScript

# Health
GET    /health                    # Health check
GET    /admin                     # Admin panel
```

### Authentication Headers

```bash
# Recommended
-H "Authorization: Bearer your-admin-token"

# Alternative
-H "X-Admin-Token: your-admin-token"
```
