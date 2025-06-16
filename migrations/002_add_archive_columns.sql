-- Migration: Add archive functionality columns
-- This migration adds archived and archived_at columns to support
-- the archive feature that allows retiring contacts from active view

-- Add archived column to track archive status
ALTER TABLE contacts ADD COLUMN archived BOOLEAN DEFAULT 0;

-- Add archived_at column to track when contact was archived
ALTER TABLE contacts ADD COLUMN archived_at TIMESTAMP;
