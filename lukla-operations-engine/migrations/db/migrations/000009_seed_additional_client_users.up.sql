-- Seed additional client portal users linked to partner clients
INSERT INTO users (
    employee_code,
    full_name,
    email,
    phone_number,
    password_hash,
    role_id,
    client_id,
    can_access_app,
    is_active
)
VALUES
    (
        'CLI-002',
        'Klien Aventura',
        'client.aventura@lukla.local',
        '081122334413',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'client'),
        (SELECT id FROM clients WHERE code = 'AVENTURA'),
        TRUE,
        TRUE
    ),
    (
        'CLI-003',
        'Klien Nordic',
        'client.nordic@lukla.local',
        '081122334414',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'client'),
        (SELECT id FROM clients WHERE code = 'NORDIC'),
        TRUE,
        TRUE
    )
ON CONFLICT (email) DO NOTHING;

