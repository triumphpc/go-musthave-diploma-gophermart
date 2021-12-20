-- +goose Up
alter table orders
    add created_at timestamptz default CURRENT_TIMESTAMP not null;

comment on column orders.created_at is 'Create date';



-- +goose Down
alter table orders drop column created_at;

