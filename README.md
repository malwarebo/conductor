# Gopay - Payment Orchestration System

[![Go Build](https://github.com/malwarebo/gopay/actions/workflows/go-build.yml/badge.svg)](https://github.com/malwarebo/gopay/actions/workflows/go-build.yml)

Gopay is an open-source payment orchestration system that supports multiple payment providers (currently Stripe and Xendit) with features for payment processing, subscriptions, and dispute management. The goal is to provide a unified interface for payments when you required more than one payment provider to fulfill your business needs. The system is built with simplicity in mind, focusing on ease of use and flexibility.

## Architecture

Architecture diagram and documentation is here: <https://github.com/malwarebo/gopay/blob/master/docs/ARCHITECTURE.md>

## Installation

1. Install dependencies:

```bash
go mod download
```

2. Set up the database:

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database and user
CREATE DATABASE gopay;
CREATE USER gopay_user WITH PASSWORD 'your_password_here';
GRANT ALL PRIVILEGES ON DATABASE gopay TO gopay_user;

# Exit psql
\q

# Run the schema migration
psql -U gopay_user -d gopay -f db/schema.sql
```

3. Configure the application:

```bash
# Copy the example config
cp config/config.example.json config/config.json

# Edit config.json with your settings:
# - Update database credentials
# - Add your Stripe API keys
# - Add your Xendit API keys
# - Adjust server settings if needed
```

## Database Setup Script

For easier setup, you can use the following script to automate the database creation process:

```bash
#!/bin/bash
# Save as setup_db.sh

DB_NAME="gopay"
DB_USER="gopay_user"
DB_PASSWORD="your_password_here"

# Create database and user
echo "Creating database and user..."
sudo -u postgres psql << EOF
CREATE DATABASE $DB_NAME;
CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
EOF

# Run the schema migration
echo "Running schema migration..."
psql -U $DB_USER -d $DB_NAME -f db/schema.sql

echo "Database setup complete!"
```

Make the script executable and run it:

```bash
chmod +x setup_db.sh
./setup_db.sh
```

## Running the Application

1. Start the server:

```bash
go run main.go
```

2. The API will be available at `http://localhost:8080`

## Docker Deployment

### Prerequisites

- Docker
- Docker Compose

### Environment Variables (optional)

Create a `.env` file in the project root with the following variables:

```
XENDIT_API_KEY=your_xendit_api_key
STRIPE_API_KEY=your_stripe_api_key
```

### Running the Application

1. Build and start the services:

```bash
docker-compose up --build
```

2. Stop the services:

```bash
docker-compose down
```

### Development with Docker

- To rebuild the image: `docker-compose build`
- To run tests in Docker: `docker-compose run --rm gopay go test ./...`

### Accessing the Application

The application will be available at `http://localhost:8080`

## API Endpoints

### Payments

- `POST /charges` - Create a new charge
- `POST /refunds` - Create a refund

### Subscriptions

- `POST /plans` - Create a subscription plan
- `GET /plans` - List all plans
- `GET /plans/:id` - Get plan details
- `PUT /plans/:id` - Update plan
- `DELETE /plans/:id` - Delete plan
- `POST /subscriptions` - Create a subscription
- `GET /subscriptions` - List subscriptions (requires customer_id parameter)
- `GET /subscriptions/:id` - Get subscription details
- `PUT /subscriptions/:id` - Update subscription
- `DELETE /subscriptions/:id` - Cancel subscription

### Disputes

- `POST /disputes` - Create a dispute
- `GET /disputes` - List disputes (requires customer_id parameter)
- `GET /disputes/:id` - Get dispute details
- `PUT /disputes/:id` - Update dispute
- `POST /disputes/:id/evidence` - Submit evidence
- `GET /disputes/stats` - Get dispute statistics

## Example Usage

1. Create a charge:

```bash
curl -X POST http://localhost:8080/charges \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 1000,
    "currency": "USD",
    "payment_method_id": "pm_card_visa",
    "customer_id": "cust_123",
    "description": "Test charge"
  }'
```

2. Create a subscription:

```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_123",
    "plan_id": "plan_123",
    "payment_method_id": "pm_card_visa"
  }'
```

3. List disputes for a customer:

```bash
curl -X GET "http://localhost:8080/disputes?customer_id=cust_123" \
  -H "Content-Type: application/json"
```

4. Submit evidence for a dispute:

```bash
curl -X POST http://localhost:8080/disputes/disp_123/evidence \
  -H "Content-Type: application/json" \
  -d '{
    "type": "customer_communication",
    "description": "Email from customer confirming receipt",
    "files": ["https://example.com/evidence1.pdf"]
  }'
```

5. Get dispute statistics:

```bash
curl -X GET http://localhost:8080/disputes/stats \
  -H "Content-Type: application/json"
```

## Project Status

1. **Phase 1 (Completed)**
   - Basic payment processing
   - Provider orchestration
   - Configuration management
   - Database integration with GORM

2. **Phase 2 (Current)**
   - Subscription management
   - Dispute handling
   - Advanced error handling
   - Improved logging

3. **Phase 3 (Future)**
   - Webhook handling
   - Event system
   - Analytics integration
   - Advanced reporting

4. **Phase 4 (Future)**
   - Additional payment providers
   - Advanced fraud detection
   - Performance optimization

## Future Considerations

1. **Integration**
   - Additional payment providers
   - Third-party services
   - Notification systems (maybe)

2. **Features**
   - Advanced reporting
   - Fraud detection
   - Real-time analytics
