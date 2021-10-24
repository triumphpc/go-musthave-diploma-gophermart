-- +goose Up
alter table orders alter column code type varchar(100) using code::varchar(100);


-- +goose Down
alter table orders alter column code type int using code::int;


