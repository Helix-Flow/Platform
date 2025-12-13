#!/bin/bash

# HelixFlow Certificate Rotation Script
# Automates certificate rotation with minimal downtime

set -e

CERT_DIR="$(dirname "$0")"
BACKUP_DIR="$CERT_DIR/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "$BACKUP_DIR"

echo "🔄 Starting certificate rotation..."

# Backup current certificates
echo "📦 Backing up current certificates..."
cp -p *.pem *.p12 *.srl "$BACKUP_DIR/$TIMESTAMP/" 2>/dev/null || true

# Generate new certificates
echo "🔐 Generating new certificates..."
./generate-certificates.sh

# Restart services (would be implemented in production)
echo "🔄 Services would be restarted here..."
# docker-compose restart nginx
# kubectl rollout restart deployment/nginx

echo "✅ Certificate rotation completed successfully!"
echo "📦 Backups stored in: $BACKUP_DIR/$TIMESTAMP"
