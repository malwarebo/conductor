# Development Guide

## Database Setup Script

Want to automate the database setup? Here's a handy script in `/scripts`:

```bash
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
```

Make it executable and run:

```bash
chmod +x setup_db.sh
./setup_db.sh
```

## Cache support

The Redis cache is ready to go and can be used to cache things like payment methods, customer info, and subscription details. Right now, the cache is set up but not actively caching. It's there for when you want to add caching to improve performance.

Here's how you might use the cache in a service:

```go
type ExampleService struct {
    store Store
    cache *cache.RedisCache
}

func NewExampleService(store Store, cache *cache.RedisCache) *ExampleService {
    return &ExampleService{
        store: store,
        cache: cache,
    }
}

func (s *ExampleService) GetItem(ctx context.Context, id string) (*Item, error) {
    cacheKey := "item:" + id
    if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
        var item Item
        if err := json.Unmarshal([]byte(cachedData), &item); err == nil {
            return &item, nil
        }
    }
    
    item, err := s.store.GetItem(ctx, id)
    if err != nil {
        return nil, err
    }
    
    if itemJSON, err := json.Marshal(item); err == nil {
        s.cache.Set(ctx, cacheKey, itemJSON)
    }
    
    return item, nil
}
```

