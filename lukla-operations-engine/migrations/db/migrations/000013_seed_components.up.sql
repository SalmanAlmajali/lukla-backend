-- Clean existing seeds if already present to ensure idempotent migration
DELETE FROM components WHERE code IN (
    'CMP-SNO-PK-01', 'CMP-SNO-PK-02', 'CMP-SNO-PK-03', 'CMP-SNO-PK-04', 'CMP-SNO-PK-05',
    'CMP-SNO-BP-01', 'CMP-SNO-BP-02', 'CMP-SNO-BP-03', 'CMP-SNO-BP-04',
    'CMP-AVE-TR-01', 'CMP-AVE-TR-02', 'CMP-AVE-TR-03',
    'CMP-AVE-DP-01', 'CMP-AVE-DP-02', 'CMP-AVE-DP-03'
);

-- Seed operational components with Standard SAM (Time Study result)
-- 1. Snøhund Arctic Expedition Parka 2026 (GARMENT)
INSERT INTO components (article_id, code, name, sequence_number, standard_sam)
VALUES
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        'CMP-SNO-PK-01',
        'Potong & Persiapan Pola Badan Parka',
        1,
        3.50
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        'CMP-SNO-PK-02',
        'Jahit Saku Kargo & Flap Tahan Air',
        2,
        4.80
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        'CMP-SNO-PK-03',
        'Pasang Resleting Depan 2-Way & Storm Flap',
        3,
        5.20
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        'CMP-SNO-PK-04',
        'Jahit Kerah Tinggi & Pasang Storm Hoodie',
        4,
        6.10
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-PK01'),
        'CMP-SNO-PK-05',
        'Assembly Lengan, Manset Velcro & Hem Bawah',
        5,
        7.00
    );

-- 2. Snøhund Alpine Summit Backpack 45L (BAG)
INSERT INTO components (article_id, code, name, sequence_number, standard_sam)
VALUES
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-BP01'),
        'CMP-SNO-BP-01',
        'Jahit Harness Bahu Ergonomis & Sternum Strap',
        1,
        6.50
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-BP01'),
        'CMP-SNO-BP-02',
        'Rakit Panel Belakang Eva Foam & Air Mesh',
        2,
        8.20
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-BP01'),
        'CMP-SNO-BP-03',
        'Jahit Kantong Samping Cordura & Kompresi Webbing',
        3,
        4.50
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-SNO-BP01'),
        'CMP-SNO-BP-04',
        'Assembly Kompartemen Utama & Roll-Top Closure',
        4,
        9.30
    );

-- 3. Aventura All-Terrain Cargo Pants (GARMENT)
INSERT INTO components (article_id, code, name, sequence_number, standard_sam)
VALUES
    (
        (SELECT id FROM articles WHERE code = 'ART-AVE-TR01'),
        'CMP-AVE-TR-01',
        'Jahit Saku Samping & Saku Kargo Paha',
        1,
        5.00
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-AVE-TR01'),
        'CMP-AVE-TR-02',
        'Pasang Resleting Fly & Ban Pinggang Elastis',
        2,
        4.20
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-AVE-TR01'),
        'CMP-AVE-TR-03',
        'Assembly Selangkangan, Jahit Kaki & Hem Bawah',
        3,
        6.30
    );

-- 4. Aventura Trail Roll-Top Daypack 22L (BAG)
INSERT INTO components (article_id, code, name, sequence_number, standard_sam)
VALUES
    (
        (SELECT id FROM articles WHERE code = 'ART-AVE-DP01'),
        'CMP-AVE-DP-01',
        'Rakit Tali Gendong Padded & Handle Atas',
        1,
        4.80
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-AVE-DP01'),
        'CMP-AVE-DP-02',
        'Pasang Laptop Sleeve Internal & Saku Depan',
        2,
        5.50
    ),
    (
        (SELECT id FROM articles WHERE code = 'ART-AVE-DP01'),
        'CMP-AVE-DP-03',
        'Assembly Bodi Utama & Roll-Top Buckle',
        3,
        7.20
    );

