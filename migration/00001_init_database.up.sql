CREATE TABLE IF NOT EXISTS user_details (
    id INTEGER PRIMARY KEY,
    first_name TEXT,
    last_name TEXT,
    email_address TEXT,
    created_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    merged_at TIMESTAMPTZ,
    parent_user_id INTEGER
);
