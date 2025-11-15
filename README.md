# Conductor - Payment Orchestration Platform

[![go build](https://github.com/malwarebo/conductor/actions/workflows/go-build.yml/badge.svg)](https://github.com/malwarebo/conductor/actions/workflows/go-build.yml)
[![docker build](https://github.com/malwarebo/conductor/actions/workflows/docker-image.yml/badge.svg)](https://github.com/malwarebo/conductor/actions/workflows/docker-image.yml)

`Conductor` is an open-source payment orchestration platform that makes handling multiple payment providers a breeze. It supports Stripe and Xendit out of the box, giving you a unified interface for payments, subscriptions, and dispute management. Perfect for when you need more than one payment provider to handle different currencies or regions.

The system includes intelligent fraud detection powered by AI that analyzes transactions in real-time before processing payments. It uses OpenAI's advanced models to identify suspicious patterns while maintaining strict privacy standards by anonymizing sensitive data. The fraud detection layer integrates seamlessly into your payment flow, automatically blocking high-risk transactions while allowing legitimate ones to proceed smoothly.

> [!NOTE]
> Want to know why I built this? Check out the story here: <https://github.com/malwarebo/conductor/blob/master/docs/PROBLEM.md>

## Architecture

Curious about how it all works under the hood? Check out the architecture docs: <https://github.com/malwarebo/conductor/blob/master/docs/ARCHITECTURE.md>

## API reference

API docs are available here: <https://github.com/malwarebo/conductor/blob/master/docs/API_REFERENCE.md>

## Quick Start

### 1. Get the dependencies

```bash
go mod download
```

### 2. Set up your database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database and user
CREATE DATABASE conductor;
CREATE USER conductor_user WITH PASSWORD 'your_password_here';
GRANT ALL PRIVILEGES ON DATABASE conductor TO conductor_user;

# Exit psql
\q

# Run the schema migration
psql -U conductor_user -d conductor -f config/db/schema.sql
```

### 3. Configure the app

**Option 1: Using Environment Variables (Recommended for Production)**

```bash
# Copy the environment template
cp env.example .env

# Edit .env with your actual values:
# - Set secure database credentials
# - Add your Stripe API keys
# - Add your Xendit API keys
# - Add your OpenAI API key for fraud detection (optional)
# - Adjust server settings if needed

# Load environment variables
export $(cat .env | xargs)
```

**Option 2: Using Configuration File (Development Only)**

```bash
# Copy the example config
cp config/config.example.json config/config.json

# Edit config.json with your settings:
# - Update database credentials
# - Add your Stripe API keys
# - Add your Xendit API keys
# - Add your OpenAI API key for fraud detection (optional)
# - Adjust server settings if needed
```

**⚠️ Security Note**: For production deployments, always use environment variables for sensitive data like API keys and database passwords. Never commit actual secrets to version control.

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

## Running the App

Start the server:

```bash
go run main.go
```

Your API will be live at `http://localhost:8080`

## Docker Deployment

### Environment Variables (optional)

Create a `.env` file in the project root:

```
XENDIT_API_KEY=your_xendit_api_key
STRIPE_API_KEY=your_stripe_api_key
```

### Running with Docker

Build and start everything:

```bash
docker-compose up --build
```

### Development with Docker

- Rebuild the image: `docker-compose build`
- Run tests in Docker: `docker-compose run --rm conductor go test ./...`

### Accessing the App

The app will be available at `http://localhost:8080`

## API Endpoints

### Payments

- `POST /v1/charges` - Create a new charge
- `POST /v1/refunds` - Create a refund

### Subscriptions

- `POST /v1/plans` - Create a subscription plan
- `GET /v1/plans` - List all plans
- `GET /v1/plans/:id` - Get plan details
- `PUT /v1/plans/:id` - Update plan
- `DELETE /v1/plans/:id` - Delete plan
- `POST /v1/subscriptions` - Create a subscription
- `GET /v1/subscriptions` - List subscriptions (requires customer_id parameter)
- `GET /v1/subscriptions/:id` - Get subscription details
- `PUT /v1/subscriptions/:id` - Update subscription
- `DELETE /v1/subscriptions/:id` - Cancel subscription

### Disputes

- `POST /v1/disputes` - Create a dispute
- `GET /v1/disputes` - List disputes (requires customer_id parameter)
- `GET /v1/disputes/:id` - Get dispute details
- `PUT /v1/disputes/:id` - Update dispute
- `POST /v1/disputes/:id/evidence` - Submit evidence
- `GET /v1/disputes/stats` - Get dispute statistics

### Fraud Detection

- `POST /v1/fraud/analyze` - Analyze transaction for fraud risk
- `GET /v1/fraud/stats` - Get fraud detection statistics

### System

- `GET /v1/health` - Health check

## Authentication

All API endpoints (except health check) require authentication using an API key. You can provide the API key in two ways:

1. **X-API-Key header** (recommended):

   ```bash
   curl -H "X-API-Key: your_api_key_here" http://localhost:8080/v1/charges
   ```

2. **Authorization Bearer header**:

   ```bash
   curl -H "Authorization: Bearer your_api_key_here" http://localhost:8080/v1/charges
   ```

**Note**: Replace `your_api_key_here` with your actual API key. For development, you can use any string with at least 10 characters.

## API Examples

Here are some real examples of how to use the API. The system automatically routes your requests to the right payment provider based on the currency!

### Creating Charges

#### Basic charge with Stripe (USD)

```bash
curl -X POST http://localhost:8080/v1/charges \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "customer_id": "cus_123456789",
    "amount": 2500,
    "currency": "USD",
    "payment_method": "pm_123456789",
    "description": "Payment for order #12345"
  }'
```

#### Charge with metadata using Xendit (IDR)

```bash
curl -X POST http://localhost:8080/v1/charges \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "customer_id": "customer_123",
    "amount": 500000,
    "currency": "IDR",
    "payment_method": "pm_xendit_123",
    "description": "Premium subscription payment",
    "metadata": {
      "order_id": "ORD-2024-001",
      "user_id": "user_456",
      "product_type": "subscription",
      "billing_cycle": "monthly"
    }
  }'
```

#### High-value charge with Stripe (EUR)

```bash
curl -X POST http://localhost:8080/v1/charges \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cus_europe_789",
    "amount": 9999,
    "currency": "EUR",
    "payment_method": "pm_europe_456",
    "description": "Annual enterprise license",
    "metadata": {
      "license_type": "enterprise",
      "duration": "annual",
      "seats": 100,
      "region": "EU"
    }
  }'
```

### Creating Refunds

#### Simple refund

```bash
curl -X POST http://localhost:8080/v1/refunds \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "payment_id": "ch_123456789",
    "amount": 2500,
    "currency": "USD",
    "reason": "Customer requested refund"
  }'
```

#### Partial refund with metadata

```bash
curl -X POST http://localhost:8080/v1/refunds \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "ch_123456789",
    "amount": 1000,
    "currency": "USD",
    "reason": "Partial refund for damaged item",
    "metadata": {
      "refund_type": "partial",
      "damage_reported": true,
      "customer_service_agent": "agent_123"
    }
  }'
```

### Managing Subscriptions

#### Create a subscription plan

```bash
curl -X POST http://localhost:8080/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Premium Plan",
    "description": "Premium features with priority support",
    "amount": 2999,
    "currency": "USD",
    "billing_period": "monthly",
    "pricing_type": "fixed",
    "trial_days": 7,
    "features": ["priority_support", "advanced_analytics", "api_access"]
  }'
```

#### Create a subscription

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cus_123456789",
    "plan_id": "plan_premium_001",
    "quantity": 1,
    "trial_days": 7,
    "payment_method_id": "pm_123456789",
    "metadata": {
      "marketing_source": "website",
      "referral_code": "WELCOME10"
    }
  }'
```

### Handling Disputes

#### Create a dispute

```bash
curl -X POST http://localhost:8080/v1/disputes \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cus_123456789",
    "transaction_id": "ch_123456789",
    "amount": 2500,
    "currency": "USD",
    "reason": "fraudulent",
    "evidence": {
      "customer_communication": "Customer claims unauthorized charge"
    },
    "due_by": "2024-02-15T23:59:59Z"
  }'
```

#### Submit evidence for a dispute

```bash
curl -X POST http://localhost:8080/v1/disputes/disp_123/evidence \
  -H "Content-Type: application/json" \
  -d '{
    "type": "customer_communication",
    "description": "Email from customer confirming receipt",
    "files": ["https://example.com/evidence1.pdf"],
    "metadata": {
      "evidence_type": "email",
      "customer_email": "customer@example.com"
    }
  }'
```

### System Health

#### Check if the system is healthy

```bash
curl -X GET http://localhost:8080/v1/health
```

## How Currency Routing Works

The system is smart about routing your payments to the right provider:

- **Stripe**: USD, EUR, GBP (perfect for international payments)
- **Xendit**: IDR, SGD, MYR, PHP, THB, VND (great for Southeast Asia)

Just specify the currency in your request, and the system automatically picks the best provider!

## Important Notes

### Amount Format

Always use the smallest currency unit:

- **USD/EUR**: cents (1000 = $10.00)
- **IDR**: rupiah (50000 = Rp 50,000)
- **SGD**: cents (1500 = S$15.00)

### Payment Methods

Make sure you're using valid payment method IDs from your chosen provider:

- Stripe: `pm_123456789`
- Xendit: `pm_xendit_123`

### Customer IDs

Your customer IDs should match what's in your provider's system.

## Redis Cache Configuration

Conductor uses Redis for caching to make things faster and reduce load on payment providers. You can customize the Redis setup through environment variables.

### Configuration Options

| Environment Variable | Description | Default Value |
|----------------------|-------------|---------------|
| `REDIS_HOST` | Redis server hostname | `localhost` |
| `REDIS_PORT` | Redis server port | `6379` |
| `REDIS_PASSWORD` | Redis password (leave empty for no password) | `""` |
| `REDIS_DB` | Redis database index | `0` |
| `REDIS_TTL` | Default TTL for cache entries (in seconds) | `86400` (24 hours) |

### Usage

The Redis cache is ready to go and can be used to cache things like payment methods, customer info, and subscription details. This cuts down on API calls to payment providers and makes everything faster.

Right now, Redis is set up but not actively caching. It's there for when you want to add caching to improve performance.

### Example Implementation

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
