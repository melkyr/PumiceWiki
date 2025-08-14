-- migrations/005_add_category_id_to_pages_table.up.sql

ALTER TABLE pages
ADD COLUMN category_id INT,
ADD FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;
