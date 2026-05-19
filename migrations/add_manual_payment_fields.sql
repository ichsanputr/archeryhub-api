-- Migration: Add manual payment fields to payment_transactions table
-- Date: 2026-05-19
-- Description: Adds fields to support manual payment with proof upload and verification

ALTER TABLE payment_transactions
ADD COLUMN proof_url VARCHAR(500) NULL COMMENT 'URL of uploaded payment proof image',
ADD COLUMN proof_uploaded_at DATETIME NULL COMMENT 'Timestamp when proof was uploaded',
ADD COLUMN verified_by VARCHAR(36) NULL COMMENT 'UUID of organizer who verified the payment',
ADD COLUMN verified_at DATETIME NULL COMMENT 'Timestamp when payment was verified',
ADD COLUMN rejection_reason TEXT NULL COMMENT 'Reason for payment rejection';

-- Update status enum to include new statuses (if using ENUM)
-- If status is VARCHAR, this is not needed
-- ALTER TABLE payment_transactions MODIFY COLUMN status ENUM('pending', 'paid', 'expired', 'failed', 'refunded', 'awaiting_verification', 'rejected') NOT NULL DEFAULT 'pending';

-- Add index for faster queries on manual payments
CREATE INDEX idx_payment_method_status ON payment_transactions(payment_method, status);
CREATE INDEX idx_proof_uploaded_at ON payment_transactions(proof_uploaded_at);
CREATE INDEX idx_verified_by ON payment_transactions(verified_by);

-- Add foreign key for verified_by (optional, depends on your user table structure)
-- ALTER TABLE payment_transactions ADD CONSTRAINT fk_verified_by FOREIGN KEY (verified_by) REFERENCES organizations(uuid) ON DELETE SET NULL;
