-- Default password untuk user selain tailor: 'password'
-- Hash bcrypt cost 10: $2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi

INSERT INTO users (
    employee_code,
    full_name,
    email,
    phone_number,
    password_hash,
    role_id,
    can_access_app,
    is_active
)
VALUES
    -- 1. Owner / Eksekutif
    (
        'EMP-EXEC-001',
        'Budi Santoso',
        'executive@lukla.local',
        '081122334401',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'owner_executive'),
        TRUE,
        TRUE
    ),
    -- 2. Tim Sample / Adis
    (
        'EMP-SMPL-001',
        'Dewi Lestari',
        'sample@lukla.local',
        '081122334402',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'sample_adis'),
        TRUE,
        TRUE
    ),
    -- 3. Gudang / Warehouse
    (
        'EMP-WRH-001',
        'Agus Pratama',
        'warehouse@lukla.local',
        '081122334403',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'warehouse'),
        TRUE,
        TRUE
    ),
    -- 4. Kepala Produksi / Mandor
    (
        'EMP-PROD-001',
        'Bambang Supriyadi',
        'production@lukla.local',
        '081122334404',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'production_head'),
        TRUE,
        TRUE
    ),
    -- 5. QC Inspector
    (
        'EMP-QC-001',
        'Siti Handayani',
        'qc@lukla.local',
        '081122334405',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'qc_inspector'),
        TRUE,
        TRUE
    ),
    -- 6. Finance & Payroll
    (
        'EMP-FIN-001',
        'Rina Permata',
        'finance@lukla.local',
        '081122334406',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'finance_payroll'),
        TRUE,
        TRUE
    ),
    -- 7. Klien Maklon / External Client
    (
        'CLI-001',
        'Klien Snøhund',
        'client@lukla.local',
        '081122334407',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'client'),
        TRUE,
        TRUE
    ),
    -- 8. Penjahit / Tailor (5 User, tanpa password, can_access_app = FALSE)
    (
        'EMP-TLR-001',
        'Ahmad Supriatna',
        NULL,
        '081122334411',
        NULL,
        (SELECT id FROM roles WHERE code = 'tailor'),
        FALSE,
        TRUE
    ),
    (
        'EMP-TLR-002',
        'Dedi Kusnadi',
        NULL,
        '081122334412',
        NULL,
        (SELECT id FROM roles WHERE code = 'tailor'),
        FALSE,
        TRUE
    ),
    (
        'EMP-TLR-003',
        'Endang Suhendar',
        NULL,
        '081122334413',
        NULL,
        (SELECT id FROM roles WHERE code = 'tailor'),
        FALSE,
        TRUE
    ),
    (
        'EMP-TLR-004',
        'Iwan Setiawan',
        NULL,
        '081122334414',
        NULL,
        (SELECT id FROM roles WHERE code = 'tailor'),
        FALSE,
        TRUE
    ),
    (
        'EMP-TLR-005',
        'Jajang Mulyana',
        NULL,
        '081122334415',
        NULL,
        (SELECT id FROM roles WHERE code = 'tailor'),
        FALSE,
        TRUE
    );

