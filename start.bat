@echo off
echo Starting WASM-as-OS System...

REM Check if Docker is available
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo Docker not found. Please install Docker to run WASM-as-OS.
    pause
    exit /b 1
)

REM Start services with Docker Compose
echo Starting services...
docker-compose up -d

REM Wait for services to start
echo Waiting for services to initialize...
timeout /t 10 /nobreak >nul

REM Check service status
echo Checking service status...
docker-compose ps

echo.
echo WASM-as-OS is starting up!
echo.
echo Web Interface: http://localhost:3000
echo API Endpoint: http://localhost:8080
echo.
echo Default credentials:
echo Username: admin
echo Password: admin123
echo.
echo Press any key to view logs...
pause >nul

REM Show logs
docker-compose logs -f