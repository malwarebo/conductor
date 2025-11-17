# Conductor Architecture

## Overview

Conductor is a payment switch that provides a unified interface for multiple payment providers while maintaining its own state and data consistency.

## Current Architecture

![Conductor Architecture Diagram](/assets/conductor_arch.png)

## Architecture Components

### 1. API

- RESTful API endpoints for payments, subscriptions, disputes, and plans
- Authentication and authorization
- Rate limiting for API requests
- Request validation and error handling

### 2. Services

- **Payment Service**: Handles payment processing and refunds
- **Subscription Service**: Manages recurring billing and subscription lifecycle
- **Dispute Service**: Handles payment disputes and evidence management
- **Configuration Service**: Manages system configuration and provider settings

### 3. Stores

- Implements data access patterns using GORM
- Handles database operations and transactions
- Provides clean interfaces for services
- Manages relationships between entities

### 4. Providers

- Abstract interface for payment providers
- Provider-specific implementations
- Handles provider API communication
- Maps provider responses to internal models

### 5. Data

- PostgreSQL database with GORM as ORM
- Redis for caching and rate limiting
- Configuration storage
- Efficient querying and data retrieval

## Data Models

### Core Entities

1. **Payment**
   - Transaction details
   - Payment status tracking
   - Provider information
   - Refund history

2. **Subscription**
   - Recurring billing information
   - Subscription status
   - Plan details
   - Payment history

3. **Dispute**
   - Dispute details
   - Evidence management
   - Resolution tracking
   - Related transaction info

4. **Plan**
   - Pricing information
   - Billing intervals
   - Features and limits
   - Active status

## Implementation Details

### Database Access

- GORM for object-relational mapping
- Structured database schema
- Automated migrations
- Transaction support
- Relationship handling

### Data Flow

1. **API Request Flow**

   ```
   HTTP Request -> Handler -> Service -> Store -> Database
                         -> Service -> Payment Provider
   ```

2. **Database Operations**

   ```
   Service Layer -> Store Layer -> GORM -> PostgreSQL
   ```
