CREATE TABLE IF NOT EXISTS custom_home_links (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    open_in_new_window BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_home_links_sort_order_id
    ON custom_home_links (sort_order ASC, id ASC);
