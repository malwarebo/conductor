# Database Setup Instructions

This document explains how to set up the PostgreSQL database for the Conductor payment orchestration system. We use GORM as our ORM for database operations.

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
CREATE DATABASE conductor;

-- Create a new user
CREATE USER conductor_user WITH PASSWORD 'your_password_here';

-- Grant privileges to the user
GRANT ALL PRIVILEGES ON DATABASE conductor TO conductor_user;
```

2. Connect to the new database:

```bash
psql -U conductor_user -d conductor
```

3. Run the schema migration:

```bash
# From the project root directory
psql -U conductor_user -d conductor -f db/schema.sql
```

## Environment Configuration

Set up the following environment variables or update the `config/config.json` file:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=conductor_user
export DB_PASSWORD=your_password_here
export DB_NAME=conductor
export DB_SSLMODE=disable
```

Or update `config/config.json`:

```json
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "conductor_user",
    "password": "your_password_here",
    "dbname": "conductor",
    "sslmode": "disable"
  }
}
```

## Using GORM

We use GORM for all database operations. Here are some key features and patterns:

### Models

Our models are defined with GORM tags for better control over database schema:

```go
type Plan struct {
    ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    Name        string    `gorm:"not null"`
    Amount      int64     `gorm:"not null"`
    Currency    string    `gorm:"not null"`
    Metadata    JSON      `gorm:"type:jsonb"`
    CreatedAt   time.Time `gorm:"autoCreateTime"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}
```

### Common Operations

1. Create a record:
```go
db.Create(&plan)
```

2. Find a record:
```go
var plan Plan
db.First(&plan, "id = ?", id)
```

3. Update a record:
```go
db.Save(&plan)
```

4. Delete a record:
```go
db.Delete(&plan)
```

5. Query with conditions:
```go
var plans []Plan
db.Where("active = ?", true).Find(&plans)
```

### Working with Relationships

GORM automatically handles relationships using the `Preload` function:

```go
// Load subscription with its plan
var subscription Subscription
db.Preload("Plan").First(&subscription, "id = ?", id)
```

### Transactions

Use transactions for operations that need to be atomic:

```go
err := db.Transaction(func(tx *gorm.DB) error {
    // Create subscription
    if err := tx.Create(&subscription).Error; err != nil {
        return err
    }
    
    // Update plan usage
    if err := tx.Model(&plan).Update("usage_count", gorm.Expr("usage_count + ?", 1)).Error; err != nil {
        return err
    }
    
    return nil
})
```

## Verify Setup

To verify the setup:

1. Connect to the database:
```bash
psql -U conductor_user -d conductor
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

1. **Connection Issues**:
   - Check PostgreSQL is running:
   ```bash
   sudo service postgresql status
   ```
   - Verify connection settings in pg_hba.conf
   - Ensure correct password in environment variables

2. **Migration Issues**:
   - Ensure UUID extension is enabled
   - Check database user has necessary permissions
   - Look for constraint violations in existing data

3. **Performance Issues**:
   - Use appropriate indexes (already set up in schema)
   - Enable GORM's debug mode to see generated SQL:
   ```go
   db.Debug().Where(...).Find(&result)
   ```

## Database Maintenance

1. **Backup the database**:
```bash
pg_dump -U conductor_user -d conductor > backup.sql
```

2. **Restore from backup**:
```bash
psql -U conductor_user -d conductor < backup.sql
```

3. **Reset the database**:
```bash
# Drop and recreate the database
dropdb -U postgres conductor
createdb -U postgres conductor
psql -U conductor_user -d conductor -f db/schema.sql
```

## Schema Updates

When updating the schema:

1. Create a new migration file with a timestamp prefix
2. Test the migration on a development database
3. Back up the production database before applying changes
4. Apply the migration during a maintenance window

For help or issues, please open a GitHub issue or contact the development team.
