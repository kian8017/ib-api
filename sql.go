package main

const SQL_CREATE_TABLES = `
CREATE TABLE IF NOT EXISTS message (
	id SERIAL PRIMARY KEY,
	content TEXT
);

CREATE TABLE IF NOT EXISTS could_bes (
	id SERIAL PRIMARY KEY,
	key TEXT NOT NULL,
	val TEXT
);

CREATE TABLE IF NOT EXISTS replacements (
	id SERIAL PRIMARY KEY,
	key TEXT NOT NULL,
	val TEXT
);

CREATE TABLE IF NOT EXISTS locations (
	id SERIAL PRIMARY KEY,
	abbr TEXT NOT NULL,
	name TEXT NOT NULL,
	is_language BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS related_locations (
	id SERIAL PRIMARY KEY,
	location_id INTEGER REFERENCES locations,
	related_id INTEGER REFERENCES locations,
	sort INTEGER,
	UNIQUE(location_id, related_id)
);

CREATE TABLE IF NOT EXISTS data_searches (
	id BIGSERIAL PRIMARY KEY,
	user_id UUID,
	search_type INTEGER,
	time TIMESTAMP,
	duration INTEGER,
	num_returned INTEGER,
	error TEXT,
	query_raw TEXT,
	query_processed TEXT,
	query_location INTEGER REFERENCES locations,
	query_type CHAR
);
`
