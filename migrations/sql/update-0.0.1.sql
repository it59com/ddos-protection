ALTER TABLE ip_weights ADD CONSTRAINT unique_user_ip UNIQUE (user_id, ip);
