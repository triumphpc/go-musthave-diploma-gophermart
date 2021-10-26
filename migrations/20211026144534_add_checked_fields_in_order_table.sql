-- +goose Up
alter table orders
    add is_check_done bool default false not null;

comment on column orders.is_check_done is 'Order checked in loyal machine';

alter table orders
    add check_attempts smallint default 0 not null;

comment on column orders.check_attempts is 'Check attempt count';

create index orders_code_is_check_done_index
    on orders (code, is_check_done);





-- +goose Down
drop index orders_code_is_check_done_index;

alter table orders drop column is_check_done;