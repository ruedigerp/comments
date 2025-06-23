# --- stop.sh - Alle Umgebungen stoppen ---
#!/bin/bash

echo "🛑 Stopping Comment System..."

# Stop development
if [ -f "docker-compose.dev.yml" ]; then
    echo "📦 Stopping development environment..."
    docker-compose -f docker-compose.dev.yml down -v
fi

# Stop production
if [ -f "docker-compose.yml" ]; then
    echo "📦 Stopping production environment..."
    docker-compose down -v
fi

echo "🧹 Cleaning up Docker resources..."
docker system prune -f

echo "✅ All services stopped and cleaned up"

# --- logs.sh - Logs anzeigen ---
#!/bin/bash

if [ "$1" = "dev" ]; then
    echo "📋 Development Logs:"
    docker-compose -f docker-compose.dev.yml logs -f
elif [ "$1" = "prod" ]; then
    echo "📋 Production Logs:"
    docker-compose logs -f
else
    echo "Usage: $0 [dev|prod]"
    echo "  $0 dev   - Show development logs"
    echo "  $0 prod  - Show production logs"
fi
