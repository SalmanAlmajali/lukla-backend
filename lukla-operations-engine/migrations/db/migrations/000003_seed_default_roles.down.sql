DELETE FROM roles
WHERE code IN (
    'owner_executive',
    'sample_adis',
    'warehouse',
    'production_head',
    'tailor',
    'qc_inspector',
    'finance_payroll',
    'client'
);

