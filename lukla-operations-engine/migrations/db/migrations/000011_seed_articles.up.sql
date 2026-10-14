-- Seed default articles for SNOHUND, AVENTURA, and NORDIC
INSERT INTO articles (client_id, code, title, category, description)
VALUES
    (
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        'ART-SNO-PK01',
        'Snøhund Arctic Expedition Parka 2026',
        'GARMENT',
        'Heavy-duty 3-layer weatherproof technical parka with storm hood and taped seams'
    ),
    (
        (SELECT id FROM clients WHERE code = 'SNOHUND'),
        'ART-SNO-BP01',
        'Snøhund Alpine Summit Backpack 45L',
        'BAG',
        'Ultralight Cordura 500D mountaineering backpack with ergonomic harness system'
    ),
    (
        (SELECT id FROM clients WHERE code = 'AVENTURA'),
        'ART-AVE-TR01',
        'Aventura All-Terrain Cargo Pants',
        'GARMENT',
        'Quick-dry water-repellent ripstop cargo pants for tropical trekking'
    ),
    (
        (SELECT id FROM clients WHERE code = 'AVENTURA'),
        'ART-AVE-DP01',
        'Aventura Trail Roll-Top Daypack 22L',
        'BAG',
        'Compact daily pack with roll-top closure and laptop compartment'
    ),
    (
        (SELECT id FROM clients WHERE code = 'NORDIC'),
        'ART-NOR-WJ01',
        'Nordic Ultralight Windbreaker Jacket',
        'GARMENT',
        'Windproof breathable running jacket with reflective accents'
    )
ON CONFLICT (code) DO NOTHING;

