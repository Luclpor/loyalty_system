create table if not exists loyalty_system.balance
(
    id          integer primary key,
    user_id uuid not null,
    point decimal default null,

    created_at  timestamp with time zone not null,
                              created_by  varchar(256) not null,
    updated_at  timestamp with time zone,
                              updated_by  varchar(256),
    constraint loyalty_system_order_user_id FOREIGN KEY (user_id) REFERENCES loyalty_system.user (Id) ON DELETE cascade
    );

drop trigger if exists tiub_loyalty_system_balance_audit on loyalty_system.balance;
create trigger tiub_loyalty_system_balance_audit
    before insert or update
                         on loyalty_system.balance
                         for each row
                         execute procedure audit_table();

create table if not exists loyalty_system.history_balance_operation
(
    id          integer primary key,
    user_id uuid not null,
    is_positive_transaction boolean not null,
    amount_transaction_point decimal default 0.0,
    balance_points decimal not null,

    created_at  timestamp with time zone not null,
    created_by  varchar(256) not null,
    updated_at  timestamp with time zone,
                              updated_by  varchar(256),
    constraint loyalty_system_order_user_id FOREIGN KEY (user_id) REFERENCES loyalty_system.user (Id) ON DELETE cascade
    );

drop trigger if exists tiub_history_balance_operation_audit on loyalty_system.history_balance_operation;
create trigger tiub_history_balance_operation_audit
    before insert or update
                         on loyalty_system.history_balance_operation
                         for each row
                         execute procedure audit_table();