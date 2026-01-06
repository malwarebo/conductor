<p align="center">
  <img src="assets/logo.svg" alt="Conductor Logo" width="80" height="80">
</p>

# Conductor - a smart payment switch

[![go build](https://github.com/malwarebo/conductor/actions/workflows/go-build.yml/badge.svg)](https://github.com/malwarebo/conductor/actions/workflows/go-build.yml)
[![docker build](https://github.com/malwarebo/conductor/actions/workflows/docker-image.yml/badge.svg)](https://github.com/malwarebo/conductor/actions/workflows/docker-image.yml)

`Conductor` is an open-source payment switch that simplifies handling multiple payment providers. It supports Stripe, Xendit, and Razorpay, giving you a unified interface for payments, subscriptions, and dispute management. Perfect for when you need more than one payment provider to handle different currencies or regions.

The system includes an `experimental` fraud detection with AI that analyzes transactions in real-time before processing payments. It uses OpenAI's LLM models to identify suspicious patterns while maintaining strict privacy standards by anonymizing sensitive data. The fraud detection layer integrates easily into your payment flow, automatically trying to block high-risk transactions while allowing legitimate ones to proceed smoothly.

> [!TIP]
> Why I'm building this? Read: <https://github.com/malwarebo/conductor/blob/master/docs/PROBLEM.md>
> 
> Architecture diagram: <https://github.com/malwarebo/conductor/blob/master/docs/ARCHITECTURE.md>
>
> API docs: <https://github.com/malwarebo/conductor/blob/master/docs/API_REFERENCE.md>

## Setup

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

> [!TIP]
> Want to automate the database setup? See the [Development Guide](docs/DEVELOPMENT.md#database-setup-script) for a handy script.

### 3. Configure the app

**Option 1: Using Environment Variables (Recommended for Production)**

```bash
# Copy the environment template
cp env.example .env

# Edit .env with your actual values:
# - Set secure database credentials
# - Add your Stripe API keys
# - Add your Xendit API keys
# - Add your Razorpay API keys
# - Add your OpenAI API key for fraud detection (experimentation and optional)
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
# - Add your Razorpay API keys
# - Add your OpenAI API key for fraud detection (experimental and optional)
# - Adjust server settings if needed
```

**⚠️ Security Note**: For production deployments, it is advised to use environment variables for API keys and database passwords.

## Running the App

Start the server:

```bash
go run main.go
```

Your API will be live at `http://localhost:8080`

## Docker setup

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

## How Currency Routing Works

The system is smart about routing your payments to the right provider:

- **Stripe**: USD, EUR, GBP (perfect for international payments)
- **Xendit**: IDR, SGD, MYR, PHP, THB, VND (great for Southeast Asia)
- **Razorpay**: INR (optimized for India with UPI and Netbanking support)

Just specify the currency in your request, and the system automatically picks the best provider.

> [!TIP]
> For more details on smart routing, see the [Smart Routing Guide](docs/SMART_ROUTING.md).

## Documentation

| Document | Description |
|----------|-------------|
| [API Reference](docs/API_REFERENCE.md) | Endpoints, examples, authentication |
| [Architecture](docs/ARCHITECTURE.md) | System design and diagrams |
| [Smart Routing](docs/SMART_ROUTING.md) | How currency routing works |
| [Fraud Detection](docs/FRAUD_DETECTION.md) | AI-powered fraud prevention |
| [Security Guide](docs/SECURITY_GUIDE.md) | Security best practices |
| [Development Guide](docs/DEVELOPMENT.md) | Database scripts, caching, dev tips |
