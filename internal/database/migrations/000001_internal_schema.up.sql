create table if not exists usr(
    id bigserial primary key ,
    username varchar(250) not null,
    created_at timestamp default now()
);

create table if not exists telegram(
    id bigserial primary key,
    username varchar(255),
    telegram_id bigint not null unique ,
    user_id bigint references usr(id) on delete cascade unique
);

create table if not exists wallet_ton(
    id bigserial primary key ,
    name varchar(255),
    addr varchar(256) not null ,
    user_id bigint references usr(id) on delete cascade unique
);

create table if not exists pool (
    id bigserial primary key ,
    owner_id bigint references usr(id) on delete cascade ,
    reserve numeric(19, 9) check ( reserve > 0 )not null ,
    jetton_wallet varchar(256) not null ,
    reward int check ( reward > 0 ) not null,
    period int check (period > 0) not null,
    is_active bool default false
);

create table if not exists stake (
    id bigserial primary key ,
    user_id bigint references usr(id) on delete cascade not null,
    pool_id bigint references pool(id) on delete cascade not null ,
    amount numeric(19, 9) not null check ( amount > 0 )
)