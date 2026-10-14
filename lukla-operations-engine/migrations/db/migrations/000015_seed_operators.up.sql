INSERT INTO operators (
    operator_code,
    name,
    phone,
    join_date,
    status,
    default_line,
    skill_tier,
    user_id
)
VALUES
    -- 1. Penjahit senior Line 1 (Linked to user EMP-TLR-001)
    (
        'JH-001',
        'Ahmad Supriatna',
        '081122334411',
        '2025-01-15',
        'ACTIVE',
        'LINE-1',
        'SENIOR',
        (SELECT id FROM users WHERE employee_code = 'EMP-TLR-001')
    ),
    -- 2. Penjahit medium Line 1 (Linked to user EMP-TLR-002)
    (
        'JH-002',
        'Dedi Kusnadi',
        '081122334412',
        '2025-06-01',
        'ACTIVE',
        'LINE-1',
        'MEDIUM',
        (SELECT id FROM users WHERE employee_code = 'EMP-TLR-002')
    ),
    -- 3. Penjahit senior Line 2 (Linked to user EMP-TLR-003)
    (
        'JH-003',
        'Endang Suhendar',
        '081122334413',
        '2025-03-10',
        'ACTIVE',
        'LINE-2',
        'SENIOR',
        (SELECT id FROM users WHERE employee_code = 'EMP-TLR-003')
    ),
    -- 4. Penjahit Line 2 sedang cuti / ON_LEAVE (Linked to user EMP-TLR-004)
    (
        'JH-004',
        'Iwan Setiawan',
        '081122334414',
        '2025-08-20',
        'ON_LEAVE',
        'LINE-2',
        'MEDIUM',
        (SELECT id FROM users WHERE employee_code = 'EMP-TLR-004')
    ),
    -- 5. Penjahit nonaktif / INACTIVE (Linked to user EMP-TLR-005)
    (
        'JH-005',
        'Jajang Mulyana',
        '081122334415',
        '2025-11-01',
        'INACTIVE',
        'LINE-1',
        'JUNIOR',
        (SELECT id FROM users WHERE employee_code = 'EMP-TLR-005')
    ),
    -- 6. Penjahit wanita medium Line 2 (Direct headcount tanpa login app)
    (
        'JH-006',
        'Siti Rahmawati',
        '081244556677',
        '2026-02-01',
        'ACTIVE',
        'LINE-2',
        'MEDIUM',
        NULL
    ),
    -- 7. Penjahit senior Line 1 (Direct headcount)
    (
        'JH-007',
        'Asep Saepudin',
        '081311223344',
        '2026-01-10',
        'ACTIVE',
        'LINE-1',
        'SENIOR',
        NULL
    ),
    -- 8. Penjahit baru masuk di tengah minggu (Skenario Uji SRS 2: Payroll mid-week joiner)
    (
        'JH-008',
        'Rahmat Hidayat',
        '081399887766',
        '2026-10-14',
        'ACTIVE',
        'LINE-2',
        'JUNIOR',
        NULL
    ),
    -- 9. Operator floating / cadangan multi-line tanpa default line tetap
    (
        'JH-009',
        'Ujang Koswara',
        '081388776655',
        '2026-03-01',
        'ACTIVE',
        NULL,
        'MEDIUM',
        NULL
    );

