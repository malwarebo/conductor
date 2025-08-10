-- Migration: Add fraud analysis results table
-- Date: 2025-08-10
-- Description: Create table for storing fraud detection analysis results

-- Create fraud analysis results table
CREATE TABLE IF NOT EXISTS fraud_analysis_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    transaction_amount DECIMAL(10,2) NOT NULL,
    billing_country VARCHAR(3) NOT NULL,
    shipping_country VARCHAR(3) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    transaction_velocity INTEGER NOT NULL,
    is_fraudulent BOOLEAN NOT NULL,
    fraud_score INTEGER NOT NULL CHECK (fraud_score >= 0 AND fraud_score <= 100),
    reason TEXT NOT NULL,
    allow BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for fraud analysis results
CREATE INDEX IF NOT EXISTS idx_fraud_analysis_transaction_id ON fraud_analysis_results(transaction_id);
CREATE INDEX IF NOT EXISTS idx_fraud_analysis_user_id ON fraud_analysis_results(user_id);
CREATE INDEX IF NOT EXISTS idx_fraud_analysis_created_at ON fraud_analysis_results(created_at);
CREATE INDEX IF NOT EXISTS idx_fraud_analysis_is_fraudulent ON fraud_analysis_results(is_fraudulent);

-- Create trigger for fraud analysis results updated_at (if the function exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at_column') THEN
        DROP TRIGGER IF EXISTS update_fraud_analysis_results_updated_at ON fraud_analysis_results;
        CREATE TRIGGER update_fraud_analysis_results_updated_at
            BEFORE UPDATE ON fraud_analysis_results
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END
$$;
