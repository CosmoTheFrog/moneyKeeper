-- Обычные валюты, добавленные пользователем (CAD, CNY...): курс берётся у Нацбанка Грузии, а не у биржи.
ALTER TABLE assets ADD COLUMN fiat BOOLEAN NOT NULL DEFAULT false;
