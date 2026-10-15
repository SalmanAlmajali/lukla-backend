-- Migration 000033 down: Drop Factory Overhead Tables (BPMN 09)

DROP TABLE IF EXISTS overhead_allocations CASCADE;
DROP TABLE IF EXISTS overhead_expenses CASCADE;
DROP TABLE IF EXISTS overhead_periods CASCADE;
DROP TABLE IF EXISTS overhead_categories CASCADE;

