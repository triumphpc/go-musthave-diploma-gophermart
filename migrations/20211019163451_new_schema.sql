-- +goose Up
create table users
(
    id serial not null
        constraint users_pk
            primary key,
    login varchar(100),
    password varchar(255)
);

comment on table users is 'Table with users';

comment on column users.id is 'Primary identifier';

comment on column users.login is 'Login of user';

create unique index users_login_uindex
    on users (login);



-- +goose Down
drop table users;
