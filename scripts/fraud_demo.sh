#!/bin/bash

# GoPay Fraud Detection Demo Script
# This script demonstrates the fraud detection functionality

echo "GoPay Fraud Detection Demo"
echo "=================================="

BASE_URL="http://localhost:8080/api/v1"
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
START_DATE="2025-08-10T00:00:00Z"
END_DATE="2025-08-10T23:59:59Z"

echo
echo "📊 Testing Fraud Detection System..."
echo

# Test 1: Low Risk Transaction
echo "1️⃣ Testing Low Risk Transaction..."
RESPONSE1=$(curl -s -X POST $BASE_URL/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_demo_low_001",
    "user_id": "user_demo_001",
    "transaction_amount": 75.50,
    "billing_country": "US",
    "shipping_country": "US",
    "ip_address": "192.168.1.100",
    "transaction_velocity": 2
  }')

echo "Response: $RESPONSE1"
echo

# Test 2: High Risk Transaction
echo "2️⃣ Testing High Risk Transaction..."
RESPONSE2=$(curl -s -X POST $BASE_URL/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_demo_high_001",
    "user_id": "user_demo_002",
    "transaction_amount": 2500.00,
    "billing_country": "US",
    "shipping_country": "NG",
    "ip_address": "197.211.56.14",
    "transaction_velocity": 12
  }')

echo "Response: $RESPONSE2"
echo

# Test 3: Medium Risk Transaction
echo "3️⃣ Testing Medium Risk Transaction..."
RESPONSE3=$(curl -s -X POST $BASE_URL/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_demo_medium_001",
    "user_id": "user_demo_003",
    "transaction_amount": 450.00,
    "billing_country": "CA",
    "shipping_country": "US",
    "ip_address": "142.103.12.55",
    "transaction_velocity": 4
  }')

echo "Response: $RESPONSE3"
echo

# Wait a moment for database to process
sleep 2

# Test Statistics
echo "📈 Testing Statistics Endpoint..."
STATS_RESPONSE=$(curl -s "$BASE_URL/fraud/stats?start_date=$START_DATE&end_date=$END_DATE")
echo "Statistics Response: $STATS_RESPONSE"
echo

echo "✅ Demo Complete!"
echo
echo "🔍 Summary:"
echo "- Tested 3 different risk levels of transactions"
echo "- Retrieved fraud statistics for the day"
echo "- All responses logged to database for analysis"
echo
echo "💡 Next Steps:"
echo "- Check the fraud_analysis_results table in your database"
echo "- Monitor the fraud detection rates over time"
echo "- Adjust thresholds based on your business requirements"
