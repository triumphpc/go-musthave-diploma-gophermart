-- +goose Up
alter table orders
    add repeat_at timestamp;

comment on column orders.repeat_at is 'Time for repeat check task';



-- +goose Down
alter table orders drop column repeat_at;

