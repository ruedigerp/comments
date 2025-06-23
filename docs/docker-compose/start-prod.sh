# --- start-prod.sh - Produktion ---
#!/bin/bash
set -e

echo "🚀 Starting Comment System (Production Mode)"
echo "============================================="

# Generate secure token if not set
if [ -z "$ADMIN_TOKEN" ]; then
    export ADMIN_TOKEN=$(openssl rand -hex 32)
    echo "🔑 Generated Admin Token: $ADMIN_TOKEN"
    echo "💡 Save this token! You'll need it to access the admin panel."
    echo ""
fi

# Create .env file for production
cat > .env << EOF
ADMIN_TOKEN=$ADMIN_TOKEN
PUBLIC_API_URL=http://localhost:8080/api/comments
DOMAIN=localhost
VERSION=1.0.0
STAGE=production
EOF

echo "📝 Created .env file with secure token"

# Clean up and start
echo "🧹 Cleaning up existing containers..."
docker-compose down -v

echo "🔨 Building and starting services..."
docker-compose up --build -d

echo "⏳ Waiting for services to be ready..."
sleep 15

# Health check
echo "🏥 Health check..."
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Application is healthy!"
else
    echo "❌ Application health check failed"
    echo "📋 Checking logs..."
    docker-compose logs comment-api
    exit 1
fi

echo ""
echo "🎉 Production environment ready!"
echo ""
echo "🌐 Application: http://localhost:8080"
echo "🔑 Admin Token: $ADMIN_TOKEN"
echo "🎛️  Admin Panel: http://localhost:8080/admin?token=$ADMIN_TOKEN"
echo "📦 Widget URL: http://localhost:8080/js/comment-widget.js"
echo ""

