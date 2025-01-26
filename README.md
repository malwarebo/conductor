# Gopay - Payment Orchestration System

Gopay is an open-source payment orchestration system that supports multiple payment providers (Stripe and Xendit) with features for payment processing, subscriptions, and dispute management.

Architecture diagram and documentation is here: https://github.com/malwarebo/gopay/blob/master/docs/ARCHITECTURE.md

## Features

- **Multi-provider Support**
  - Stripe and Xendit integration
  - Automatic provider failover
  - Easy addition of new providers

- **Payment Processing**
  - Charge processing
  - Refund handling
  - Transaction management

- **Subscription Management**
  - Plan creation and management
  - Subscription lifecycle handling
  - Trial period support
  - Automatic billing

- **Dispute Handling**
  - Dispute creation and management
  - Evidence submission
  - Status tracking
  - Dispute statistics

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher
- Stripe account and API key
- Xendit account and API key

## Installation

1. Clone the repository:
```bash
git clone https://github.com/malwarebo/gopay.git
cd gopay
```

2. Install dependencies:
```bash
go mod download
```

3. Set up the database:
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

4. Configure the application:
```bash
# Copy the example config
cp config/config.example.json config/config.json

# Edit config.json with your settings:
# - Update database credentials
# - Add your Stripe API keys
# - Add your Xendit API keys
# - Adjust server settings if needed
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

### Environment Variables
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
- `GET /charges/:id` - Get charge details
- `POST /refunds` - Create a refund

### Subscriptions
- `POST /plans` - Create a subscription plan
- `GET /plans` - List all plans
- `GET /plans/:id` - Get plan details
- `POST /subscriptions` - Create a subscription
- `GET /subscriptions/:id` - Get subscription details
- `PUT /subscriptions/:id` - Update subscription
- `DELETE /subscriptions/:id` - Cancel subscription

### Disputes
- `POST /disputes` - Create a dispute
- `GET /disputes/:id` - Get dispute details
- `PUT /disputes/:id` - Update dispute
- `POST /disputes/:id/evidence` - Submit evidence

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

## Development

### Running Tests
```bash
go test ./...
```

### Code Style
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
