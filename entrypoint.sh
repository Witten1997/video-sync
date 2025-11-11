#!/bin/bash
set -e

# 初始化 PostgreSQL（如果未初始化）
if [ ! -d "$PGDATA" ] || [ ! -f "$PGDATA/PG_VERSION" ]; then
    echo "初始化 PostgreSQL 数据库..."
    mkdir -p "$PGDATA"
    chown -R postgres:postgres /var/lib/postgresql
    chmod 700 "$PGDATA"

    su - postgres -c "/usr/lib/postgresql/$POSTGRES_VERSION/bin/initdb -D $PGDATA -E UTF8 --locale=zh_CN.UTF-8"

    # 配置 PostgreSQL 监听本地
    echo "host all all 127.0.0.1/32 trust" >> "$PGDATA/pg_hba.conf"
    echo "host all all ::1/128 trust" >> "$PGDATA/pg_hba.conf"
    echo "local all all trust" >> "$PGDATA/pg_hba.conf"

    # 启动 PostgreSQL
    su - postgres -c "/usr/lib/postgresql/$POSTGRES_VERSION/bin/pg_ctl -D $PGDATA -l /var/log/postgresql.log start"

    # 等待 PostgreSQL 启动
    sleep 5

    # 创建数据库和用户
    su - postgres -c "psql -c \"CREATE USER ${POSTGRES_USER:-bili_sync} WITH PASSWORD '${POSTGRES_PASSWORD:-bili_sync}';\""
    su - postgres -c "psql -c \"CREATE DATABASE ${POSTGRES_DB:-bili_sync} OWNER ${POSTGRES_USER:-bili_sync};\""
    su - postgres -c "psql -d ${POSTGRES_DB:-bili_sync} -f /app/bili-sync-schema.sql" || true

    # 停止 PostgreSQL（supervisor 会重新启动）
    su - postgres -c "/usr/lib/postgresql/$POSTGRES_VERSION/bin/pg_ctl -D $PGDATA stop"

    echo "PostgreSQL 初始化完成"
fi

# 更新配置文件中的数据库连接信息
sed -i "s/host: \"localhost\"/host: \"127.0.0.1\"/g" /app/configs/config.yaml
sed -i "s/user: \"bili_sync\"/user: \"${POSTGRES_USER:-bili_sync}\"/g" /app/configs/config.yaml
sed -i "s/password: \"your_password\"/password: \"${POSTGRES_PASSWORD:-bili_sync}\"/g" /app/configs/config.yaml
sed -i "s/dbname: \"bili_sync\"/dbname: \"${POSTGRES_DB:-bili_sync}\"/g" /app/configs/config.yaml

# 确保目录权限正确
chown -R postgres:postgres /var/lib/postgresql
chmod -R 755 /downloads /metadata /var/log/bili-sync

# 启动 supervisor
echo "启动所有服务..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
