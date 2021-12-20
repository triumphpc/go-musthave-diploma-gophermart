-- +goose Up
alter table users
    add points int default 0 not null;


-- +goose Down
alter table users drop column points;

