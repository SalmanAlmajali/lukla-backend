-- Seed clients untuk demonstrasi pagination dan master data klien maklon
INSERT INTO clients (code, name, contact, address, client_type)
VALUES
    ('AVENTURA', 'Aventura Outdoor Gear', 'contact@aventura.id', 'Jl. Riau No. 45, Bandung', 'B2B'),
    ('NORDIC', 'Nordic Trail Apparel', 'hello@nordictrail.co', 'Jl. Setiabudi No. 120, Bandung', 'B2B'),
    ('EQUATOR', 'Equator Pack & Bag', 'info@equatorbags.com', 'Jl. Soekarno-Hatta No. 510, Bandung', 'B2B'),
    ('ALPEN', 'Alpen Technical Wear', 'procurement@alpenwear.id', 'Jl. Dago No. 88, Bandung', 'B2B'),
    ('SUMMIT', 'Summit Supply Co.', 'admin@summitsupply.co', 'Jl. Buah Batu No. 15, Bandung', 'B2B'),
    ('KEMBARA', 'Kembara Adventure Goods', 'halo@kembara.id', 'Jl. Cihampelas No. 201, Bandung', 'B2B'),
    ('RIMBA', 'Rimba Tactical Gear', 'ops@rimbatactical.id', 'Kawasan Industri Cimahi, Bandung Barat', 'B2B'),
    ('BORNEO', 'Borneo Overland Project', 'partnership@borneo-op.com', 'Jl. LLRE Martadinata No. 70, Bandung', 'B2B'),
    ('JAVA_EXP', 'Java Expedition Wear', 'sales@javaexpedition.com', 'Jl. Gatot Subroto No. 33, Bandung', 'B2B'),
    ('TERRA', 'Terra Urban Packs', 'support@terrapaks.id', 'Jl. Dipati Ukur No. 42, Bandung', 'B2B'),
    ('HIGHLAND', 'Highland Canvas Co.', 'order@highlandcanvas.com', 'Kopo Sayati, Bandung', 'B2B')
ON CONFLICT (code) DO NOTHING;

