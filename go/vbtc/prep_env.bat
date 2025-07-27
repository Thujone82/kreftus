@echo off
echo.
echo ** Preparing Go Environment **
echo.

echo 1. Setting environment to bypass Go proxy for this session...
set GOPROXY=direct

echo.
echo 2. Cleaning the Go module cache...
go clean -modcache

echo.
echo 3. Tidying project modules (will re-download dependencies)...
go mod tidy

echo.
echo ** Environment is ready. **
echo You can now build the application in this same terminal window.
echo.
