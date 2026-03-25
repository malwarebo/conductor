# Smart Routing

Conductor uses a deterministic smart routing engine that selects the optimal payment provider based on real-time metrics, historical performance, and configurable rules.

## How It Works

The routing engine scores each provider using weighted factors:

| Factor | Weight | Description |
|--------|--------|-------------|
| Success Rate | 35% | Real-time success rate from rolling window |
| Cost | 20% | Transaction cost as percentage |
| BIN Performance | 20% | Historical success rate for card issuer |
| Latency | 10% | Average response time |
| Health | 10% | Circuit breaker state |
| Volume | 5% | Distribution against targets |

## Currency Routing

Default provider preferences by currency:

| Currency | Primary Provider |
|----------|-----------------|
| USD, EUR, GBP, CAD | Stripe |
| IDR, PHP, VND, THB, MYR | Xendit |
| INR | Razorpay |
| HKD, CNY, AUD, NZD, JPY, KRW | Airwallex |

## Circuit Breakers

Each provider has a circuit breaker that automatically stops traffic when failures exceed thresholds:

- **Closed**: Normal operation
- **Open**: Provider disabled after 5 consecutive failures
- **Half-Open**: Testing recovery after 30s timeout

```go
config := circuitbreaker.Config{
    FailureThreshold:    5,
    SuccessThreshold:    3,
    Timeout:             30 * time.Second,
    RollingWindowSize:   60 * time.Second,
    MinRequestsInWindow: 10,
}
```

## BIN/IIN Routing

The engine tracks success rates per card BIN (first 6 digits) per provider:

```
HDFC cards (BIN 438628):
  - Razorpay: 99.1% success
  - Stripe: 94.2% success
  → Routes to Razorpay
```

## Smart Retry

Failed payments automatically retry on fallback providers:

1. Classify error (soft decline vs hard decline)
2. If retryable, try next provider with exponential backoff
3. Max 3 attempts across different providers

```go
config := routing.RetryConfig{
    MaxAttempts:       3,
    Strategy:          RetryStrategySmartFallback,
    InitialDelay:      100 * time.Millisecond,
    MaxDelay:          2 * time.Second,
    BackoffMultiplier: 2.0,
}
```

## Merchant Configuration

Per-merchant routing preferences:

```go
config := &models.MerchantRoutingConfig{
    MerchantID:         "merchant_123",
    PreferredProviders:  []string{"stripe", "razorpay"},
    ExcludedProviders:   []string{"xendit"},
    MinSuccessRate:      0.9,
    MaxCostPercent:      0.035,
    EnableSmartRouting:  true,
    EnableRetry:         true,
    MaxRetryAttempts:    2,
    VolumeTargets: map[string]float64{
        "stripe":   0.6,
        "razorpay": 0.4,
    },
}
```

## Routing Rules

Custom rules with conditions:

```go
rule := &models.RoutingRule{
    Name:           "high_value_stripe",
    Priority:       100,
    Enabled:        true,
    TargetProvider: "stripe",
    Weight:         0.2,
    Conditions: models.RoutingConditions{
        Currencies: []string{"USD", "EUR"},
        MinAmount:  ptr(1000.0),
        CustomerSegments: []string{"enterprise"},
    },
}
```

## Monitoring

Get routing stats from the multi-provider selector:

```go
stats := providerSelector.GetRoutingStats()
healthy := providerSelector.GetHealthyProviders()
isHealthy := providerSelector.IsProviderHealthy("stripe")
```

Stats include:
- Circuit breaker states per provider
- Success rates (1m, 5m, 1h windows)
- Latency percentiles
- Volume distribution
- Decision metrics

## Configuration

Enable smart routing when creating the provider selector:

```go
config := providers.MultiProviderConfig{
    EnableSmartRouting: true,
    BINStore:          binStore,
    MerchantStore:     merchantStore,
    RuleStore:         ruleStore,
    RoutingConfig:     routing.DefaultConfig(),
    RetryConfig:       routing.DefaultRetryConfig(),
}

selector := providers.CreateMultiProviderSelectorWithConfig(
    providers,
    mappingStore,
    config,
)
```
