-- Migration 000029 Down: Drop QC Inspection Tables
DROP TABLE IF EXISTS qc_inspection_defects CASCADE;
DROP TABLE IF EXISTS qc_inspections CASCADE;
DROP TABLE IF EXISTS qc_defect_categories CASCADE;

