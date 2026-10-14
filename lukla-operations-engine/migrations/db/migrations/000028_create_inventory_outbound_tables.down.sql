-- Migration 000028 Down: Drop Inventory Outbound, Blocked Attempts, and Cutting Scraps tables

DROP TABLE IF EXISTS cutting_scraps CASCADE;
DROP TABLE IF EXISTS inventory_blocked_attempts CASCADE;
DROP TABLE IF EXISTS inventory_issue_items CASCADE;
DROP TABLE IF EXISTS inventory_issues CASCADE;

