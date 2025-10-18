-- Add partner_name field for married users
ALTER TABLE users 
ADD COLUMN partner_name varchar;
