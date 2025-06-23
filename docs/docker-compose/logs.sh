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
