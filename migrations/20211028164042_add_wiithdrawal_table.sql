-- +goose Up
create table withdrawals
(
    id serial not null
        constraint withdrawals_pk
            primary key,
    user_id int not null,
    order_id int not null,
    points int not null
);

comment on table withdrawals is 'Users withdrawal';

comment on column withdrawals.id is 'Identifier of withdrawal';

comment on column withdrawals.user_id is 'Identifier of user';

comment on column withdrawals.order_id is 'Identifier of user order';

comment on column withdrawals.points is 'Sum of withdrawal';

alter table withdrawals
    add status smallint default 0 not null;

comment on column withdrawals.status is 'Status of withdraw';

alter table withdrawals
    add processed_at timestamptz;

comment on column withdrawals.processed_at is 'Processed time';




-- +goose Down
drop table withdrawals;


