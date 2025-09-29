# Smart Routing

GoPay includes AI-powered intelligent routing that uses OpenAI to make optimal provider selection decisions based on transaction context, historical performance, and cost optimization.

## API Endpoints

### 1. Smart Provider Selection

```
POST /api/v1/routing/select
```

**Request Body:**
```json
{
  "currency": "USD",
  "amount": 150.00,
  "country": "US",
  "customer_segment": "premium",
  "transaction_type": "one_time",
  "merchant_id": "merchant_123",
  "customer_id": "customer_456",
  "ip_address": "192.168.1.1",
  "metadata": {
    "product_category": "electronics",
    "payment_method": "credit_card"
  }
}
```

**Response:**
```json
{
  "recommended_provider": "stripe",
  "confidence_score": 92,
  "reasoning": "Stripe offers the best success rate (98.5%) and competitive pricing (2.9%) for USD transactions in the US market",
  "alternative_provider": "xendit",
  "estimated_success_rate": 0.985,
  "estimated_cost": 4.65,
  "routing_time_ms": 150,
  "cache_hit": false,
  "fallback_used": false
}
```

### 2. Provider Statistics

```
GET /api/v1/routing/stats
```

**Response:**
```json
{
  "stripe": {
    "success_rate": 0.985,
    "avg_response_time_ms": 450,
    "cost_per_transaction": 0.029,
    "total_transactions": 15000,
    "failed_transactions": 225,
    "last_updated": "2024-01-15T10:30:00Z"
  },
  "xendit": {
    "success_rate": 0.972,
    "avg_response_time_ms": 380,
    "cost_per_transaction": 0.025,
    "total_transactions": 8500,
    "failed_transactions": 238,
    "last_updated": "2024-01-15T10:30:00Z"
  }
}
```

### 3. Routing Metrics

```
GET /api/v1/routing/metrics?start_date=2024-01-01T00:00:00Z&end_date=2024-01-15T23:59:59Z
```

**Response:**
```json
{
  "total_decisions": 2500,
  "cache_hit_rate": 0.75,
  "avg_confidence_score": 87.5,
  "success_rate": 0.96,
  "avg_response_time_ms": 180,
  "cost_savings": 1250.50,
  "provider_distribution": {
    "stripe": 1600,
    "xendit": 900
  }
}
```

### 4. Routing Configuration

```
GET /api/v1/routing/config
PUT /api/v1/routing/config
```

**Configuration:**
```json
{
  "enable_ai_routing": true,
  "cache_ttl_seconds": 3600,
  "min_confidence_score": 70,
  "fallback_provider": "stripe",
  "enable_cost_optimization": true,
  "enable_success_rate_optimization": true
}
```

## Usage Examples

### Basic Routing Request

```bash
curl -X POST http://localhost:8080/api/v1/routing/select \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "currency": "USD",
    "amount": 99.99,
    "country": "US",
    "customer_segment": "standard",
    "transaction_type": "one_time"
  }'
```

### High-Value Transaction

```bash
curl -X POST http://localhost:8080/api/v1/routing/select \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "currency": "EUR",
    "amount": 2500.00,
    "country": "DE",
    "customer_segment": "enterprise",
    "transaction_type": "subscription",
    "metadata": {
      "subscription_plan": "enterprise",
      "billing_cycle": "annual"
    }
  }'
```

### Southeast Asian Transaction

```bash
curl -X POST http://localhost:8080/api/v1/routing/select \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "currency": "IDR",
    "amount": 500000,
    "country": "ID",
    "customer_segment": "standard",
    "transaction_type": "one_time"
  }'
```

## AI Routing Logic

The AI considers multiple factors when making routing decisions:

### 1. Currency Optimization

- **USD/EUR/GBP**: Prefers Stripe (global expertise)
- **IDR/SGD/MYR/PHP/THB/VND**: Prefers Xendit (regional expertise)

### 2. Performance Metrics

- **Success Rate**: Historical transaction success rates
- **Response Time**: Average API response times
- **Cost Efficiency**: Processing fees and transaction costs

### 3. Transaction Context

- **Amount**: High-value transactions may prefer different providers
- **Customer Segment**: Premium customers get priority routing
- **Transaction Type**: Subscriptions vs one-time payments
- **Geographic Location**: Local provider preferences

### 4. Real-time Factors

- **Time of Day**: Peak vs off-peak hours
- **Provider Health**: Current availability and performance
- **Load Balancing**: Distribute load across providers

## Fallback Logic

When OpenAI is unavailable, the system uses rule-based routing:

```go
// Currency-based fallback
switch currency {
case "USD", "EUR", "GBP":
    return "stripe"
case "IDR", "SGD", "MYR", "PHP", "THB", "VND":
    return "xendit"
default:
    return "stripe" // Default fallback
}
```

## Caching Strategy

- **Cache Key**: Based on currency, amount, country, and customer segment
- **TTL**: 1 hour (configurable)
- **Cache Hit Rate**: Typically 75-80% for similar transactions
- **Performance**: Sub-millisecond cache lookups

## Monitoring and Analytics

### Key Metrics Tracked

1. **Routing Decisions**: Total decisions made
2. **Cache Performance**: Hit rates and response times
3. **AI Confidence**: Average confidence scores
4. **Success Rates**: Transaction success by provider
5. **Cost Savings**: Estimated savings from optimal routing
6. **Provider Distribution**: Usage patterns across providers

### Performance Benchmarks

- **AI Routing**: 150-300ms average response time
- **Cache Hit**: < 5ms response time
- **Fallback Routing**: < 10ms response time
- **Confidence Score**: 85-95% average

## Configuration Options

### Environment Variables

```bash
# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key_here

# Routing Configuration
ROUTING_CACHE_TTL=3600
ROUTING_MIN_CONFIDENCE=70
ROUTING_FALLBACK_PROVIDER=stripe
```

### Runtime Configuration

Use the `/api/v1/routing/config` endpoint to adjust settings:

- **Enable/Disable AI Routing**: Toggle AI-powered decisions
- **Cache TTL**: Adjust caching duration
- **Confidence Threshold**: Minimum confidence for AI decisions
- **Fallback Provider**: Default provider when AI fails
- **Optimization Flags**: Enable cost or success rate optimization

## Integration with Payment Flow

The intelligent routing integrates seamlessly with the existing payment flow:

```go
// 1. Get routing recommendation
routingResp, err := routingService.SelectOptimalProvider(ctx, routingReq)

// 2. Use recommended provider for payment
paymentReq := &models.ChargeRequest{
    Currency:      routingReq.Currency,
    Amount:        routingReq.Amount,
    // ... other fields
}

// 3. Route to optimal provider
chargeResp, err := providerSelector.Charge(ctx, paymentReq)
```

## Troubleshooting

### Common Issues

1. **Low Confidence Scores**: Check provider statistics and transaction data
2. **High Response Times**: Monitor OpenAI API performance
3. **Cache Misses**: Review cache key generation logic
4. **Fallback Usage**: Check OpenAI API availability and configuration

### Debugging

Enable detailed logging to troubleshoot routing decisions:

```bash
# Check routing logs
curl -X GET http://localhost:8080/api/v1/routing/metrics

# Monitor provider stats
curl -X GET http://localhost:8080/api/v1/routing/stats
```
