-- Initialize the CTF database

USE ctf;

CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user'
);

CREATE TABLE secrets (
    id INT AUTO_INCREMENT PRIMARY KEY,
    secret_name VARCHAR(100) NOT NULL,
    secret_value TEXT NOT NULL
);

-- Insert some users
INSERT INTO users (username, password, role) VALUES 
    ('guest', 'guest123', 'user'),
    ('admin', 'sup3rs3cr3t_4dm1n_p4ss!', 'admin');

-- Insert the flag as a secret
INSERT INTO secrets (secret_name, secret_value) VALUES 
    ('flag', 'FLAG{c0mp0s3_1nst4nc3s_r0ck!}'),
    ('admin_note', 'Remember to change the admin password!');
