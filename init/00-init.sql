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
        -- IPv4 OR IPv6/64 CIDR
        version ENUM('4', '6') NOT NULL,
        ip_bin BINARY(16) NOT NULL UNIQUE,
        rule_type ENUM ('whitelist', 'blacklist') NOT NULL,
        create_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        note VARCHAR(255),
        INDEX idx_rule_version (version, ip_bin)
    );

USE ktauth;

INSERT INTO
    ip (version, ip_bin, rule_type)
VALUES
    ("4", UNHEX("00000000000000000000FFFF7F000001"), "whitelist");