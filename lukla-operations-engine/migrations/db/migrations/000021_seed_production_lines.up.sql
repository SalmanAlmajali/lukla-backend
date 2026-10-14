INSERT INTO lines (
    code,
    name,
    max_capacity,
    supervisor_id,
    status
)
VALUES
    (
        'LINE-01',
        'Line Jahit Parka & Jaket Tebal',
        24,
        (SELECT id FROM users WHERE email = 'production@lukla.local'),
        'ACTIVE'
    ),
    (
        'LINE-02',
        'Line Jahit Celana & Tas Kargo',
        22,
        (SELECT id FROM users WHERE email = 'production@lukla.local'),
        'ACTIVE'
    ),
    (
        'LINE-03',
        'Line Cadangan & Finishing',
        24,
        NULL,
        'ACTIVE'
    )
ON CONFLICT (code) DO NOTHING;

