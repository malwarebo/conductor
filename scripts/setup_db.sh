#!/bin/bash

set -e

DB_NAME="conductor"
DB_USER="conductor_user"
DB_PASSWORD="your_password_here"

echo "Creating database and user..."
if sudo -u postgres psql << EOF
CREATE DATABASE $DB_NAME;
CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
EOF
then
    echo "Database and user created successfully"
else
    echo "Failed to create database and user"
    exit 1
fi

echo ""
echo "Running schema migration..."
if psql -U $DB_USER -d $DB_NAME -f config/db/schema.sql 2>&1 | tee /tmp/schema_output.log | grep -q "ERROR"; then
    echo ""
    echo "Schema migration failed with errors:"
    grep "ERROR" /tmp/schema_output.log
    rm -f /tmp/schema_output.log
    exit 1
else
    echo "Schema migration completed successfully"
    rm -f /tmp/schema_output.log
fi

echo ""
echo "Database setup complete!"
