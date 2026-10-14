UPDATE users
SET client_id = NULL
WHERE email = 'client@lukla.local';

DELETE FROM clients WHERE code = 'SNOHUND';

