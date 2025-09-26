@echo off
setlocal
if "%GK_DB_DSN%"=="" set GK_DB_DSN=postgres://postgres:pass@localhost:5432/keeper?sslmode=disable
if "%GK_GRPC_ADDR%"=="" set GK_GRPC_ADDR=:8090
if "%GK_RUN_MIGRATIONS%"=="" set GK_RUN_MIGRATIONS=true
set GK_DB_DSN=postgres://postgres:pass@localhost:5432/keeper?sslmode=disable
set GK_RUN_MIGRATIONS=true
set GK_TLS_ENABLE=true
set GK_TLS_CERT_PATH=C:\Users\tony\Downloads\gophkeeper\tls\server.crt
set GK_TLS_KEY_PATH=C:\Users\tony\Downloads\gophkeeper\tls\server.key
echo [SERVER] %GK_GRPC_ADDR%  DB=%GK_DB_DSN%
go run .\cmd\gk-server