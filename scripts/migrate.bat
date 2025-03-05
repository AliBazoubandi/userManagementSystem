@echo off
set DB_USER=alibazoubandi
set DB_PASSWORD=575980899598
set DB_HOST=localhost
set DB_PORT=5432
set DB_NAME=myDataBase
set SSL_MODE=disable

:: Check if golang-migrate is installed
where migrate >nul 2>nul
if %errorlevel% neq 0 (
    echo golang-migrate not found. Installing...
    go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest
)

:: Run migrations
echo Running database migrations...
migrate -path ./migrations_using_go -database "postgres://%DB_USER%:%DB_PASSWORD%@%DB_HOST%:%DB_PORT%/%DB_NAME%?sslmode=%SSL_MODE%" up

echo Migration completed successfully.
pause
