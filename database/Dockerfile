# ==============================================================================
# Dockerfile for PostgreSQL 16 Database with Automatic Schema Initialization
# ==============================================================================

FROM postgres:16-alpine

# Set Indonesian Western Time (WIB) default
ENV TZ=Asia/Jakarta

# Copy DDL schema to initialization directory
# PostgreSQL automatically executes all .sql files in /docker-entrypoint-initdb.d on first container boot
COPY schema.sql /docker-entrypoint-initdb.d/01_schema.sql

# Expose standard PostgreSQL port
EXPOSE 5432

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=5 \
    CMD pg_isready -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-umkm_pos} || exit 1
