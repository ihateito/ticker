CREATE TABLE ticks
(
    timestamp BIGINT     NOT NULL,
    symbol    varchar(6) NOT NULL,
    bid       float      NOT NULL,
    ask       float      NOT NULL
);

