-- Seed default client: Snøhund Brand
INSERT INTO clients (code, name, contact, address, client_type)
VALUES (
    'SNOHUND',
    'Snøhund Brand',
    'ops@snohund.co',
    'Ciwidey, Bandung, Jawa Barat',
    'INTERNAL'
);

-- Tautkan akun klien maklon ke data client Snøhund
UPDATE users
SET client_id = (SELECT id FROM clients WHERE code = 'SNOHUND')
WHERE email = 'client@lukla.local';

