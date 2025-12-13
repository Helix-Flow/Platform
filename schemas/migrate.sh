#!/bin/bash
# Database migration script for HelixFlow
# Handles schema versioning and data migrations

set -e

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-helixflow}
DB_USER=${DB_USER:-postgres}
MIGRATION_DIR="./migrations"

# Create migrations directory
mkdir -p "${MIGRATION_DIR}"

# Generate new migration file
generate_migration() {
    local description=$1
    local timestamp=$(date +%Y%m%d%H%M%S)
    local filename="${timestamp}_${description}.sql"

    cat > "${MIGRATION_DIR}/${filename}" << EOF
-- Migration: ${description}
-- Created: $(date)
-- Version: ${timestamp}

BEGIN;

-- Add your migration SQL here

COMMIT;
EOF

    echo "Created migration: ${filename}"
}

# Run migrations
run_migrations() {
    echo "Running database migrations..."

    # Get list of applied migrations
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c "
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR(255) PRIMARY KEY,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
    " > /dev/null

    # Get applied versions
    APPLIED=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c "
        SELECT version FROM schema_migrations ORDER BY version;
    " | tr -d ' ')

    # Apply pending migrations
    for migration in "${MIGRATION_DIR}"/*.sql; do
        if [ -f "$migration" ]; then
            filename=$(basename "$migration")
            version="${filename%%_*}"

            if ! echo "$APPLIED" | grep -q "^${version}$"; then
                echo "Applying migration: ${filename}"
                psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -f "$migration"

                # Record migration
                psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c "
                    INSERT INTO schema_migrations (version) VALUES ('${version}');
                " > /dev/null

                echo "Migration applied successfully"
            fi
        fi
    done

    echo "All migrations applied"
}

# Rollback last migration
rollback_migration() {
    echo "Rolling back last migration..."

    LAST_VERSION=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c "
        SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1;
    " | tr -d ' ')

    if [ -n "$LAST_VERSION" ]; then
        # Find rollback file
        ROLLBACK_FILE="${MIGRATION_DIR}/${LAST_VERSION}_rollback.sql"
        if [ -f "$ROLLBACK_FILE" ]; then
            echo "Rolling back migration: ${LAST_VERSION}"
            psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -f "$ROLLBACK_FILE"

            # Remove migration record
            psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c "
                DELETE FROM schema_migrations WHERE version = '${LAST_VERSION}';
            " > /dev/null

            echo "Migration rolled back successfully"
        else
            echo "No rollback file found for migration ${LAST_VERSION}"
        fi
    else
        echo "No migrations to rollback"
    fi
}

# Show migration status
status() {
    echo "Migration Status:"
    echo "=================="

    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c "
        SELECT version, applied_at
        FROM schema_migrations
        ORDER BY applied_at;
    "
}

# Main command handling
case "${1:-help}" in
    generate)
        if [ -z "$2" ]; then
            echo "Usage: $0 generate <description>"
            exit 1
        fi
        generate_migration "$2"
        ;;
    run)
        run_migrations
        ;;
    rollback)
        rollback_migration
        ;;
    status)
        status
        ;;
    *)
        echo "Usage: $0 {generate <desc>|run|rollback|status}"
        echo ""
        echo "Commands:"
        echo "  generate <desc>  Generate new migration file"
        echo "  run              Apply pending migrations"
        echo "  rollback         Rollback last migration"
        echo "  status           Show migration status"
        ;;
esac