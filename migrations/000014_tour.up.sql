-- Показано ли пользователю обучение при первом входе; кнопка в «Счетах» сбрасывает флаг и показывает его снова.
ALTER TABLE users ADD COLUMN tour_seen BOOLEAN NOT NULL DEFAULT false;
