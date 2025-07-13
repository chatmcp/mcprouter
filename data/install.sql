CREATE TABLE IF NOT EXISTS servers (
    uuid VARCHAR(255) NOT NULL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    name VARCHAR(255) NOT NULL,
    author_name VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT,
    server_key VARCHAR(255) UNIQUE NOT NULL,
    server_url VARCHAR(255) NOT NULL,
    config_name VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS tools (
    uuid VARCHAR(255) NOT NULL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    name VARCHAR(255) NOT NULL,
    server_key VARCHAR(255) NOT NULL,
    description TEXT,
    input_schema TEXT,
    raw TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uni_server_name ON servers (name, author_name);
CREATE UNIQUE INDEX IF NOT EXISTS uni_tool_name ON tools (name, server_key);