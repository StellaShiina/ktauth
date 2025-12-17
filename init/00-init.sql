USE ktauth;

CREATE TABLE
    user (
        uuid CHAR(64) PRIMARY KEY,
        name VARCHAR(64) NOT NULL UNIQUE,
        password_hash CHAR(60) NOT NULL,
        email VARCHAR(255) UNIQUE
    );

CREATE TABLE
    ip (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        cidr VARCHAR(43) NOT NULL UNIQUE,
        rule_type ENUM ('whitelist', 'blacklist') NOT NULL,
        note VARCHAR(255),
        create_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        INDEX idx_rule_type (rule_type)
    );