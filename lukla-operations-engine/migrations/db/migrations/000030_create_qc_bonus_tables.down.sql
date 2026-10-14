-- Migration 000030 Down: Drop QC Bonus Tables
DROP TABLE IF EXISTS qc_bonus_distributions CASCADE;
DROP TABLE IF EXISTS qc_bonus_periods CASCADE;
DROP TABLE IF EXISTS qc_bonus_recipients CASCADE;
DROP TABLE IF EXISTS qc_bonus_rate_history CASCADE;
DROP TABLE IF EXISTS qc_bonus_configs CASCADE;

