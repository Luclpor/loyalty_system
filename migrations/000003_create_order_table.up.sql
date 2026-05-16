create table if not exists loyalty_system.order
(
    id          varchar(112) primary key,
    user_id uuid not null,
    status varchar(12),

    created_at  timestamp with time zone not null,
                              created_by  varchar(256) not null,
    updated_at  timestamp with time zone,
                              updated_by  varchar(256),
    constraint loyalty_system_order_user_id FOREIGN KEY (user_id) REFERENCES loyalty_system.user (Id) ON DELETE cascade
    );

drop trigger if exists tiub_loyalty_system_order_audit on loyalty_system.order;
create trigger tiub_loyalty_system_order_audit
    before insert or update
                         on loyalty_system.order
                         for each row
                         execute procedure audit_table();