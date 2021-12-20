-- +goose Up
create table if not exists orders
(
    id           serial  not null
        constraint orders_pk
            primary key,
    user_id      integer not null,
    code         integer not null,
    check_status smallint default 0
);

comment on table orders is 'Orders ids list for check';

comment on column orders.id is 'Unique udentifier';

comment on column orders.user_id is 'User identifier';

comment on column orders.code is 'Order number';

comment on column orders.check_status is 'Processing status';

alter table orders
    owner to postgres;

create unique index if not exists orders_code_uindex
    on orders (code);

alter table users
    add auth_token varchar(100);

comment on column users.auth_token is 'User auth token';

create unique index users_auth_token_uindex
    on users (auth_token);




-- +goose Down
drop table orders;

drop index users_auth_token_uindex;

alter table users drop column auth_token;
