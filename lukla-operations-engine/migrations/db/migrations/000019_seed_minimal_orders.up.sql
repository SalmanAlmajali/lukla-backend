INSERT INTO orders (
    order_number,
    client_id,
    article_id,
    target_quantity,
    status,
    notes,
    start_date,
    target_date
)
VALUES
    (
        'PO-2026-10-001',
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        1000,
        'IN_PRODUCTION',
        'Pesanan Ekspor Snohund Arctic Parka Batch 1',
        CURRENT_DATE,
        CURRENT_DATE + INTERVAL '30 days'
    ),
    (
        'PO-2026-10-002',
        (SELECT id FROM clients WHERE code = 'AVENTURA'),
        (SELECT id FROM articles WHERE code = 'ART-AVE-TR01'),
        500,
        'IN_PRODUCTION',
        'Pesanan Celana Kargo Aventura Q4',
        CURRENT_DATE,
        CURRENT_DATE + INTERVAL '21 days'
    )
ON CONFLICT (order_number) DO NOTHING;

