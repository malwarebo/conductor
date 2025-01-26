# Database Setup Instructions

This document explains how to set up the PostgreSQL database for the Gopay payment orchestration system.

## Prerequisites

1. PostgreSQL 12 or higher installed on your system
2. psql command-line tool
3. Superuser access to PostgreSQL

## Setup Steps

1. Create a new database and user:

```sql
-- Connect to PostgreSQL as superuser
psql -U postgres

-- Create a new database
CREATE DATABASE gopay;

-- Create a new user
CREATE USER gopay_user WITH PASSWORD 'your_password_here';

-- Grant privileges to the user
GRANT ALL PRIVILEGES ON DATABASE gopay TO gopay_user;
```

2. Connect to the new database:

```bash
psql -U gopay_user -d gopay
```

3. Run the schema migration:

```bash
# From the project root directory
psql -U gopay_user -d gopay -f db/schema.sql
```

## Environment Configuration

Set up the following environment variables or update the `config/config.json` file:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=gopay_user
export DB_PASSWORD=your_password_here
export DB_NAME=gopay
export DB_SSLMODE=disable
```

Or update `config/config.json`:

```json
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "gopay_user",
    "password": "your_password_here",
    "dbname": "gopay",
    "sslmode": "disable"
  }
}
```

## Verify Setup

To verify the setup:

1. Connect to the database:
```bash
psql -U gopay_user -d gopay
```

2. List all tables:
```sql
\dt
```

You should see the following tables:
- plans
- subscriptions
- payments
- refunds
- disputes
- dispute_evidence

## Common Issues

1. **Permission Denied**:
   - Ensure the user has proper permissions:
   ```sql
   GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO gopay_user;
   GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO gopay_user;
   ```

2. **Database Connection Failed**:
   - Check if PostgreSQL is running:
   ```bash
   sudo service postgresql status
   ```
   - Verify connection settings in pg_hba.conf
   - Ensure the correct password is being used

3. **Schema Migration Failed**:
   - Check if the database exists and is accessible
   - Ensure you have the latest schema.sql file
   - Look for error messages in the migration output

## Database Maintenance

1. **Backup the database**:
```bash
pg_dump -U gopay_user -d gopay > backup.sql
```

2. **Restore from backup**:
```bash
psql -U gopay_user -d gopay < backup.sql
```

3. **Reset the database**:
```bash
# Drop and recreate the database
dropdb -U postgres gopay
createdb -U postgres gopay
psql -U gopay_user -d gopay -f db/schema.sql
```

## Schema Updates

When updating the schema:

1. Create a new migration file with a timestamp prefix
2. Test the migration on a development database
3. Back up the production database before applying changes
4. Apply the migration during a maintenance window

For help or issues, please open a GitHub issue or contact the development team.
