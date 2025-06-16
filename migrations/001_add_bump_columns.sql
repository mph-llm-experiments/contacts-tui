-- Migration: Add bump functionality columns
-- This migration adds last_bump_date and bump_count columns to support
-- the bump feature that allows resetting contact dates without actual contact

-- Add last_bump_date column to track when contact was last bumped
ALTER TABLE contacts ADD COLUMN last_bump_date TIMESTAMP;

-- Add bump_count column to track how many times contact has been bumped
ALTER TABLE contacts ADD COLUMN bump_count INTEGER DEFAULT 0;