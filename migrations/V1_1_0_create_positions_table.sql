CREATE TABLE IF NOT EXISTS positions
(
    id_         UUID PRIMARY KEY,
    price_open  float,
    is_bay      BOOLEAN,
    symbol      VARCHAR(60),
    price_close float,
    bay_by      VARCHAR(60),
    user_id     UUID,
    stop_loss   float,
    take_profit float

);
CREATE OR REPLACE FUNCTION insert_position() RETURNS TRIGGER AS
$$
DECLARE
    channel text := TG_ARGV[0];
BEGIN
        PERFORM (
        with pos(id_, price_open, is_bay, symbol, price_close, bay_by, user_id, stop_loss, take_profit) as
                 (
                     select new.id_,
                            new.price_open,
                            new.is_bay,
                            new.symbol,
                            new.price_close,
                            new.bay_by,
                            new.user_id,
                            new.stop_loss,
                            new.take_profit
                 )
        select pg_notify(channel, row_to_json(pos)::text)
        from pos
    );
    Return NULL;
END;

$$ LANGUAGE plpgsql;

CREATE TRIGGER insert_position_notify
    AFTER INSERT OR UPDATE
    ON positions
    FOR EACH ROW
EXECUTE PROCEDURE insert_position('positions');