DELETE FROM users
WHERE employee_code IN (
    'EMP-EXEC-001',
    'EMP-SMPL-001',
    'EMP-WRH-001',
    'EMP-PROD-001',
    'EMP-QC-001',
    'EMP-FIN-001',
    'CLI-001',
    'EMP-TLR-001',
    'EMP-TLR-002',
    'EMP-TLR-003',
    'EMP-TLR-004',
    'EMP-TLR-005'
)
OR email IN (
    'executive@lukla.local',
    'sample@lukla.local',
    'warehouse@lukla.local',
    'production@lukla.local',
    'qc@lukla.local',
    'finance@lukla.local',
    'client@lukla.local'
);

