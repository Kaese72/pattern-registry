CREATE TABLE IF NOT EXISTS organizations (
    id INT PRIMARY KEY
);

ALTER TABLE patterns ADD FOREIGN KEY (owner) REFERENCES organizations(id);