CREATE TABLE patterns (
    id SERIAL PRIMARY KEY,
    pattern VARCHAR(1024) NOT NULL,
    component VARCHAR(1024) NOT NULL
)