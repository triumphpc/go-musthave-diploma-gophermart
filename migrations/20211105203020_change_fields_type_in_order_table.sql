-- +goose Up
alter table orders alter column accrual type double precision using accrual::double precision;

alter table orders alter column avail_for_withdraw type double precision using avail_for_withdraw::double precision;

alter table withdrawals alter column points type double precision using points::double precision;




-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
