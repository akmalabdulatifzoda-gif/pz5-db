-- пересоздаём БД, чтобы id начинались с 1 (можно запускать повторно)
DROP DATABASE IF EXISTS todo;
-- создать БД (если ещё нет)
CREATE DATABASE todo;
-- подключиться к ней
\c todo
-- создать таблицу
CREATE TABLE IF NOT EXISTS tasks (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  done BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- быстрый тест
INSERT INTO tasks (title) VALUES ('Первая задача из psql');
SELECT * FROM tasks;
