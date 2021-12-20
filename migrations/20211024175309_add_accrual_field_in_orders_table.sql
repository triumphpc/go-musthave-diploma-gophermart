-- +goose Up
alter table orders
    add accrual int default 0 not null;

comment on column orders.accrual is 'Accrual points';



-- +goose Down
alter table orders drop column accrual;

