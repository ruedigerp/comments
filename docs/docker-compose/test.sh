# --- test.sh - Integration Tests ---
#!/bin/bash
set -e

echo "🧪 Running Integration Tests"
echo "============================"

# Start dev environment
echo "🚀 Starting test environment..."
docker-compose -f docker-compose.dev.yml up -d

# Wait for startup
echo "⏳ Waiting for services..."
sleep 15

# Test health endpoint
echo "🏥 Testing health endpoint..."
if curl -f http://localhost:8080/health; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed"
    exit 1
fi

# Test comment creation
echo ""
echo "📝 Testing comment creation..."
RESPONSE=$(curl -s -X POST "http://localhost:8080/api/comments" \
    -H "Content-Type: application/json" \
    -d '{
        "post_id": "test-post",
        "username": "TestUser",
        "mailaddress": "test@example.com",
        "text": "Integration test comment"
    }')

if echo "$RESPONSE" | grep -q "test-post"; then
    echo "✅ Comment creation passed"
else
    echo "❌ Comment creation failed"
    echo "Response: $RESPONSE"
    exit 1
fi

# Test comment retrieval
echo ""
echo "📖 Testing comment retrieval..."
COMMENTS=$(curl -s "http://localhost:8080/api/comments?post_id=test-post")

if echo "$COMMENTS" | grep -q "TestUser"; then
    echo "✅ Comment retrieval passed"
else
    echo "❌ Comment retrieval failed"
    echo "Response: $COMMENTS"
    exit 1
fi

# Test admin endpoint
echo ""
echo "🔐 Testing admin endpoint..."
ADMIN_INFO=$(curl -s -H "Authorization: Bearer dev-token-not-for-production-12345" \
    "http://localhost:8080/api/comments/admin/info")

if echo "$ADMIN_INFO" | grep -q "total_comments"; then
    echo "✅ Admin endpoint passed"
else
    echo "❌ Admin endpoint failed"
    echo "Response: $ADMIN_INFO"
    exit 1
fi

echo ""
echo "🎉 All tests passed!"

# Cleanup
echo "🧹 Cleaning up test environment..."
docker-compose -f docker-compose.dev.yml down
