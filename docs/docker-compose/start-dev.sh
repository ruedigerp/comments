#!/bin/bash

# ===========================================
# DOCKER-COMPOSE START SCRIPTS
# ===========================================

# --- start-dev.sh - Entwicklung ---
#!/bin/bash
set -e

echo "🚀 Starting Comment System (Development Mode)"
echo "=============================================="

# Clean up any existing containers
echo "🧹 Cleaning up existing containers..."
docker-compose -f docker-compose.dev.yml down -v

# Start services
echo "🐳 Starting services..."
docker-compose -f docker-compose.dev.yml up --build -d

# Wait for services
echo "⏳ Waiting for services to be ready..."
sleep 10

# Show status
echo ""
echo "📊 Service Status:"
docker-compose -f docker-compose.dev.yml ps

echo ""
echo "🎉 Development environment ready!"
echo ""
echo "🌐 Application: http://localhost:8080"
echo "🔑 Admin Token: dev-token-not-for-production-12345"
echo "🎛️  Admin Panel: http://localhost:8080/admin?token=dev-token-not-for-production-12345"
echo "📦 Widget URL: http://localhost:8080/js/comment-widget.js"
echo "🗄️  Redis: localhost:6379"
echo ""
echo "📋 Useful commands:"
echo "  docker-compose -f docker-compose.dev.yml logs -f     # Logs anzeigen"
echo "  docker-compose -f docker-compose.dev.yml down       # Stoppen"
echo "  docker-compose -f docker-compose.dev.yml exec comment-api sh  # Shell in Container"

