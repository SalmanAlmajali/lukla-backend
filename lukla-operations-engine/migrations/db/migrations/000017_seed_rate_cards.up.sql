-- 1. Seed Rate Cards for Snøhund Arctic Expedition Parka 2026
INSERT INTO rate_cards (client_id, article_id, component_id, sewing_rate, cutting_rate, effective_date, notes)
VALUES
    (
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        (SELECT id FROM components WHERE code = 'CMP-SNO-PK-01'),
        0.00,
        1500.00,
        '2026-10-01',
        'Ongkos potong pola badan parka'
    ),
    (
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        (SELECT id FROM components WHERE code = 'CMP-SNO-PK-02'),
        4500.00,
        0.00,
        '2026-10-01',
        'Saku kargo flap ganda tahan air'
    ),
    (
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        (SELECT id FROM components WHERE code = 'CMP-SNO-PK-03'),
        5500.00,
        0.00,
        '2026-10-01',
        'Pasang resleting 2-way Aquaguard'
    ),
    (
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        (SELECT id FROM components WHERE code = 'CMP-SNO-PK-04'),
        6200.00,
        800.00,
        '2026-10-01',
        'Penyesuaian kain Cordura tebal'
    ),
    (
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        (SELECT id FROM components WHERE code = 'CMP-SNO-PK-05'),
        7000.00,
        0.00,
        '2026-10-01',
        'Assembly lengan & manset velcro'
    );

-- 2. Seed Rate Cards for Aventura Cargo Pants
INSERT INTO rate_cards (client_id, article_id, component_id, sewing_rate, cutting_rate, effective_date, notes)
VALUES
    (
        (SELECT id FROM clients WHERE code = 'AVENTURA'),
        (SELECT id FROM articles WHERE code = 'ART-AVE-TR01'),
        (SELECT id FROM components WHERE code = 'CMP-AVE-TR-01'),
        3800.00,
        600.00,
        '2026-10-01',
        'Saku samping & paha'
    ),
    (
        (SELECT id FROM clients WHERE code = 'AVENTURA'),
        (SELECT id FROM articles WHERE code = 'ART-AVE-TR01'),
        (SELECT id FROM components WHERE code = 'CMP-AVE-TR-02'),
        3200.00,
        0.00,
        '2026-10-01',
        'Fly front & ban pinggang'
    ),
    (
        (SELECT id FROM clients WHERE code = 'AVENTURA'),
        (SELECT id FROM articles WHERE code = 'ART-AVE-TR01'),
        (SELECT id FROM components WHERE code = 'CMP-AVE-TR-03'),
        4200.00,
        0.00,
        '2026-10-01',
        'Assembly inseam & hemming'
    );

-- 3. Seed initial rate card audit history for CMP-SNO-PK-04
INSERT INTO rate_card_histories (
    rate_card_id,
    old_sewing_rate,
    new_sewing_rate,
    old_cutting_rate,
    new_cutting_rate,
    changed_by_user_id,
    reason,
    created_at
)
VALUES (
    (SELECT id FROM rate_cards WHERE component_id = (SELECT id FROM components WHERE code = 'CMP-SNO-PK-04')),
    5800.00,
    6200.00,
    700.00,
    800.00,
    (SELECT id FROM users WHERE email = 'production@lukla.local'),
    'Kenaikan kompleksitas jahitan reflektif 3M pada bahu dan kerah',
    '2026-10-01 09:30:00+07'
);

