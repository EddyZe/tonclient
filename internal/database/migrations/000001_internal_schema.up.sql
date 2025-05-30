create table if not exists usr
(
    id         bigserial primary key,
    username   varchar(250) not null,
    created_at timestamp default now(),
    referer_id bigint    default null
);

create table if not exists telegram
(
    id          bigserial primary key,
    username    varchar(255),
    telegram_id bigint not null unique,
    user_id     bigint references usr (id) on delete cascade unique
);

create table if not exists wallet_ton
(
    id      bigserial primary key,
    name    varchar(255),
    addr    varchar(256) not null,
    user_id bigint references usr (id) on delete cascade unique
);

create table if not exists pool
(
    id                 bigserial primary key,
    owner_id           bigint references usr (id) on delete cascade,
    jetton_name        varchar   default 'Не указано',
    reserve            numeric(19, 9)                      not null,
    jetton_wallet      varchar(256)                        not null,
    jetton_master      varchar(256)                        not null,
    reward             int check ( reward > 0 )            not null,
    period             int check (period > 0)              not null,
    insurance_coating  int check ( insurance_coating > 0 ) not null,
    created_at         timestamp default now(),
    is_active          bool      default false,
    is_commission_paid bool      default false
);

create table if not exists stake
(
    id                     bigserial primary key,
    user_id                bigint references usr (id) on delete cascade  not null,
    pool_id                bigint references pool (id) on delete cascade not null,
    amount                 numeric(19, 9)                                not null check ( amount > 0 ),
    deposit_creation_price numeric(19, 9)                                not null,
    jetton_price_closed    numeric(19, 9)                                                               default 0 not null,
    balance                numeric(19, 9)                                not null check ( balance > 0 ) default 0,
    start_date             timestamp                                                                    default now(),
    is_active              bool                                                                         default true,
    is_insurance_paid      bool                                                                         default false,
    is_reward_paid         bool                                                                         default false,
    is_commission_paid     bool                                                                         default false
);

create table if not exists referral
(
    id               bigserial primary key,
    referrer_user_id bigint references usr (id) on delete set null,
    referral_user_id bigint references usr (id) on delete set null,
    first_stake_id   bigint references stake (id) on delete set null,
    reward_given     bool           default false,
    reward_amount    numeric(19, 9) default 0
);

create table if not exists operation
(
    id            bigserial primary key,
    user_id       bigint references usr (id) on delete cascade not null,
    num_operation int,
    name          varchar(255),
    created_at    timestamp default now(),
    description   varchar
)