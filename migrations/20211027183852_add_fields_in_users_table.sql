-- +goose Up
alter table users
    add withdrawn float default 0 not null;

comment on column users.withdrawn is 'How much user withdraw';

alter table users alter column points type float using points::float;



-- +goose Down
alter table users alter column points type int using points::int;

alter table users drop column withdrawn;
