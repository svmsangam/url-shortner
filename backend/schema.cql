-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS urlshortener 
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '1'};

-- Drop obsolete structures
DROP INDEX IF EXISTS urlshortener_urls_short_code;
DROP TABLE IF EXISTS urlshortener.urls;
DROP TABLE IF EXISTS urlshortener.device_urls;

-- Create new tables
CREATE TABLE IF NOT EXISTS urlshortener.urls (
    short_code text,
    long_url text,
    device_token text,
    created_at timestamp,
    PRIMARY KEY (short_code)
);

CREATE TABLE IF NOT EXISTS urlshortener.device_urls (
    device_token text,
    short_code text,
    long_url text,
    created_at timestamp,
    PRIMARY KEY (device_token, short_code)
);

CREATE TABLE IF NOT EXISTS urlshortener.url_clicks (
    short_code text,
    click_count counter,
    PRIMARY KEY (short_code)
);