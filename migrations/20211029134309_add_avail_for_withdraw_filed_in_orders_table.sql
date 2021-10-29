-- +goose Up
alter table orders
    add avail_for_withdraw int default 0;

comment on column orders.avail_for_withdraw is 'Available for withdraw by order';



-- +goose Down
alter table orders drop column avail_for_withdraw;

