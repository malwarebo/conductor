# Development Guide

## Database Setup

```bash
#!/bin/bash
set -e

DB_NAME="conductor"
DB_USER="conductor_user"
DB_PASSWORD="your_password_here"

sudo -u postgres psql << EOF
CREATE DATABASE $DB_NAME;
CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
EOF

psql -U $DB_USER -d $DB_NAME -f config/db/schema.sql
```

## Redis Cache

The cache infrastructure is ready for:
- Payment methods
- Customer info
- Subscription details

Usage example:

```go
type ExampleService struct {
    store Store
    cache *cache.RedisCache
}

func (s *ExampleService) GetItem(ctx context.Context, id string) (*Item, error) {
    cacheKey := "item:" + id
    if data, err := s.cache.Get(ctx, cacheKey); err == nil {
        var item Item
        if json.Unmarshal([]byte(data), &item) == nil {
            return &item, nil
        }
    }
    
    item, err := s.store.GetItem(ctx, id)
    if err != nil {
        return nil, err
    }
    
    if data, err := json.Marshal(item); err == nil {
        s.cache.Set(ctx, cacheKey, data)
    }
    
    return item, nil
}
```

## Running Tests

```bash
go test ./...
```

## Building

```bash
make build
```

## Environment Variables

Copy `env.example` to `.env` and configure:
- Database connection
- Redis connection  
- Provider API keys (Stripe, Xendit, Razorpay, Airwallex)
- JWT secret
- OpenAI key (for fraud detection)
