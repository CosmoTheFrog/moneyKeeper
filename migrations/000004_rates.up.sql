-- Курсы к лари на календарный день: цена одной целой единицы валюты в GEL.
-- Курс дня пишется один раз (первый полученный), поэтому суммы в базовой валюте не «плывут».
CREATE TABLE rates (
    day          DATE          NOT NULL,
    currency     TEXT          NOT NULL,
    gel_per_unit NUMERIC(24,10) NOT NULL CHECK (gel_per_unit > 0),
    PRIMARY KEY (day, currency)
);
