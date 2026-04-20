CREATE TABLE IF NOT EXISTS comics (
    id INT PRIMARY KEY,
    img_url TEXT NOT NULL,
    keywords TEXT[]
);