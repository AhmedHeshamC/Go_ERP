# ERPGo - Complete Setup and Usage Guide

Welcome to ERPGo! This is a comprehensive, step-by-step guide for beginners to set up, configure, and use this modern Enterprise Resource Planning system.

## 📋 Table of Contents

1. [What is ERPGo?](#what-is-erpgo)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Detailed Setup Instructions](#detailed-setup-instructions)
5. [Configuration](#configuration)
6. [Running the Application](#running-the-application)
7. [Using the API](#using-the-api)
8. [JWT Token Generation](#jwt-token-generation)
9. [Project Structure](#project-structure)
10. [Development Guide](#development-guide)
11. [Troubleshooting](#troubleshooting)
12. [FAQ](#faq)

## 🎯 What is ERPGo?

ERPGo is a modern Enterprise Resource Planning (ERP) system built with Go. Think of it as a digital headquarters for your business that helps you manage:

- **Users and Permissions** - Who can access what
- **Products** - What you sell or use
- **Inventory** - How much stock you have
- **Orders** - Customer purchases and sales
- **Customers** - Who buys from you
- **Warehouses** - Where you store your products

**Why Go?** Go is chosen for its speed, reliability, and simplicity - perfect for business applications that need to handle many users simultaneously.

## 🛠️ Prerequisites

Before you begin, make sure you have these installed on your computer:

### Required Software

1. **Go (Version 1.24 or newer)**
   - Download from: [https://golang.org/dl/](https://golang.org/dl/)
   - Verify installation: Open terminal and run `go version`
   - You should see something like: `go version go1.24.0 darwin/amd64`

2. **PostgreSQL Database**
   - Download from: [https://www.postgresql.org/download/](https://www.postgresql.org/download/)
   - Or install with Homebrew (Mac): `brew install postgresql`
   - Start the service: `brew services start postgresql`
   - Create database user: `createuser -s erpgo`

3. **Git**
   - Download from: [https://git-scm.com/downloads](https://git-scm.com/downloads)
   - Verify: `git --version`

4. **Redis (Optional but recommended)**
   - Download from: [https://redis.io/download](https://redis.io/download)
   - Or install with Homebrew: `brew install redis`
   - Start: `brew services start redis`

### Optional (but recommended)

- **Docker** - For containerized deployment
- **VS Code** - Code editor with Go extensions
- **Postman** or **Insomnia** - For API testing

## 🚀 Quick Start

If you're experienced and want to get running quickly:

```bash
# 1. Clone the repository
git clone <repository-url>
cd Go_ERP

# 2. Install dependencies
go mod download

# 3. Set up environment
cp .env.example .env
# Edit .env with your database credentials

# 4. Create database
createdb erpgo

# 5. Run the application
go run ./cmd/api

# 6. Generate a JWT token
./scripts/generate-jwt.sh -e admin@company.com -u admin -r "admin,user"
```

The API will be running at: `http://localhost:8080`

## 📚 Detailed Setup Instructions

### Step 1: Get the Code

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd Go_ERP
   ```

2. **Verify you're in the right directory**
   ```bash
   pwd
   # You should see something like: /Users/yourname/Desktop/Go_ERP
   ls
   # You should see files like go.mod, cmd/, pkg/, etc.
   ```

### Step 2: Install Go Dependencies

```bash
# Download all required Go packages
go mod download

# Verify modules are downloaded
go mod verify
```

This might take a few minutes as it downloads various packages for database connections, JWT handling, etc.

### Step 3: Set Up Your Database

1. **Start PostgreSQL**
   ```bash
   # On Mac with Homebrew
   brew services start postgresql

   # On Linux
   sudo systemctl start postgresql

   # On Windows
   # Start PostgreSQL from Services or use pgAdmin
   ```

2. **Create the database**
   ```bash
   # Connect to PostgreSQL
   psql postgres

   # Inside PostgreSQL:
   CREATE USER erpgo WITH PASSWORD 'your-secure-password';
   CREATE DATABASE erpgo OWNER erpgo;
   GRANT ALL PRIVILEGES ON DATABASE erpgo TO erpgo;
   \q  # Exit PostgreSQL
   ```

3. **Verify database connection**
   ```bash
   psql -h localhost -U erpgo -d erpgo
   # If successful, you'll see: erpgo=#
   \q  # Exit
   ```

### Step 4: Configure Environment Variables

1. **Copy the example environment file**
   ```bash
   cp .env.example .env
   ```

2. **Generate a secure JWT secret**
   ```bash
   openssl rand -base64 32
   # Copy the output - you'll need it in the next step
   ```

3. **Edit the .env file**
   ```bash
   # Using VS Code
   code .env

   # Or using nano (command line editor)
   nano .env
   ```

4. **Update these important settings in .env:**
   ```env
   # Replace the JWT_SECRET with your generated secret
   JWT_SECRET=your-generated-secret-here

   # Update database settings
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=erpgo
   DB_PASSWORD=your-secure-password
   DB_NAME=erpgo

   # Application settings
   APP_PORT=8080
   LOG_LEVEL=debug
   DEBUG_MODE=true
   ```

5. **Save and close the file**

### Step 5: Verify Your Setup

```bash
# Check if Go can compile the project
go build ./cmd/api

# If no errors, you should see a new file named 'api' in your directory
ls -la api
```

### Step 6: Run the Application

```bash
# Start the ERPGo API server
go run ./cmd/api
```

You should see output like:
```
2025/11/09 14:46:32 Starting ERPGo API server
2025/11/09 14:46:32 Server listening on :8080
2025/11/09 14:46:32 Database connection established
```

**Leave this terminal window open** - the server is now running!

### Step 7: Test the API

Open a new terminal window and test:

```bash
# Test if the server is responding
curl http://localhost:8080/health

# You should see something like:
# {"status":"ok","timestamp":"2025-11-09T14:46:32Z"}
```

## ⚙️ Configuration

### Environment Variables Explained

Your `.env` file contains these important settings:

| Variable | Description | Example |
|----------|-------------|---------|
| `JWT_SECRET` | Secret key for signing tokens | `fA3wqqvYIWU2olT8f6AnO7DYyVAMlTar5t1tT9s+htQ=` |
| `DB_HOST` | Database server address | `localhost` |
| `DB_PORT` | Database server port | `5432` |
| `DB_USER` | Database username | `erpgo` |
| `DB_PASSWORD` | Database password | `your-secure-password` |
| `DB_NAME` | Database name | `erpgo` |
| `APP_PORT` | Port for the API server | `8080` |
| `LOG_LEVEL` | Logging detail level | `debug`, `info`, `warn`, `error` |
| `DEBUG_MODE` | Enable debug features | `true` or `false` |

### Database Connection Troubles?

If you get database connection errors:

1. **Check if PostgreSQL is running:**
   ```bash
   brew services list | grep postgresql  # Mac
   sudo systemctl status postgresql       # Linux
   ```

2. **Test connection manually:**
   ```bash
   psql -h localhost -U erpgo -d erpgo
   ```

3. **Common connection strings in .env:**
   ```env
   # For local development
   DATABASE_URL=postgres://erpgo:your-password@localhost:5432/erpgo?sslmode=disable

   # For production with SSL
   DATABASE_URL=postgres://erpgo:your-password@localhost:5432/erpgo?sslmode=require
   ```

## 🏃‍♂️ Running the Application

### Development Mode

```bash
# Run with auto-restart on file changes
# Install 'air' first: go install github.com/cosmtrek/air@latest
air
```

### Production Mode

```bash
# Build the application
go build -o erpgo ./cmd/api

# Run the compiled binary
./erpgo
```

### Using Docker (Optional)

```bash
# Build Docker image
docker build -t erpgo .

# Run with Docker
docker run -p 8080:8080 --env-file .env erpgo
```

## 🔌 Using the API

### Understanding JWT Authentication

ERPGo uses JWT (JSON Web Tokens) for authentication. Here's how it works:

1. **Generate a token** with user information
2. **Include the token** in your API requests
3. **The server validates** the token and gives you access

### Step 1: Generate a JWT Token

```bash
# Generate a token for an admin user
./scripts/generate-jwt.sh -e admin@company.com -u admin -r "admin,user,manager"
```

You'll get output like:
```
ACCESS TOKEN:
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9... (long string)

Usage:
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Copy the access token** - you'll need it for API requests.

### Step 2: Make API Requests

#### Using curl (Command Line)

```bash
# Get current user info
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     http://localhost:8080/api/v1/users/me

# List all users
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     http://localhost:8080/api/v1/users

# Create a new user
curl -X POST \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "email": "newuser@company.com",
       "username": "newuser",
       "password": "SecurePassword123",
       "first_name": "John",
       "last_name": "Doe"
     }' \
     http://localhost:8080/api/v1/users
```

#### Using Postman

1. **Create a new request**
2. **Set the HTTP method** (GET, POST, PUT, DELETE)
3. **Enter the URL**: `http://localhost:8080/api/v1/users`
4. **Go to Headers tab**
5. **Add a new header**:
   - Key: `Authorization`
   - Value: `Bearer YOUR_ACCESS_TOKEN`
6. **Send the request**

### Available API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users/me` | Get current user info |
| GET | `/api/v1/users` | List all users |
| POST | `/api/v1/users` | Create new user |
| GET | `/api/v1/products` | List all products |
| POST | `/api/v1/products` | Create new product |
| GET | `/api/v1/orders` | List all orders |
| POST | `/api/v1/orders` | Create new order |
| GET | `/health` | Health check |

## 🎫 JWT Token Generation

ERPGo provides multiple ways to generate JWT tokens:

### Method 1: Shell Script (Easiest)

```bash
# Basic usage
./scripts/generate-jwt.sh

# Custom user
./scripts/generate-jwt.sh -e user@company.com -u myuser -r "user"

# Multiple roles
./scripts/generate-jwt.sh -e admin@company.com -u admin -r "admin,user,manager"

# JSON output
./scripts/generate-jwt.sh -j -e user@company.com
```

### Method 2: Go Command (Advanced)

```bash
# Generate with custom settings
go run ./cmd/generate-jwt -email dev@company.com -username dev -roles "admin,user"

# Help for all options
go run ./cmd/generate-jwt --help
```

### Token Expiration

- **Access tokens**: 15 minutes (default)
- **Refresh tokens**: 7 days (default)
- **Custom expiry**: Use `-access-expiry 1h` and `-refresh-expiry 24h`

### Using Different User Types

```bash
# Regular user
./scripts/generate-jwt.sh -e employee@company.com -u employee -r "user"

# Manager
./scripts/generate-jwt.sh -e manager@company.com -u manager -r "user,manager"

# Admin
./scripts/generate-jwt.sh -e admin@company.com -u admin -r "admin,user,manager"
```

## 📁 Project Structure

Understanding how the project is organized:

```
Go_ERP/
├── cmd/                    # Application entry points
│   ├── api/               # Main API server
│   ├── export-jwt/        # JWT export tool
│   └── generate-jwt/      # JWT generator tool
├── internal/              # Private application code
│   ├── application/       # Business logic layer
│   │   └── services/      # Service implementations
│   ├── domain/           # Business entities and rules
│   │   ├── users/        # User domain
│   │   ├── products/     # Product domain
│   │   └── orders/       # Order domain
│   ├── infrastructure/   # External concerns
│   └── interfaces/       # External interfaces
│       └── http/         # HTTP handlers
├── pkg/                   # Reusable packages
│   ├── auth/             # Authentication & JWT
│   ├── config/           # Configuration
│   ├── database/         # Database utilities
│   └── logger/           # Logging utilities
├── scripts/              # Helper scripts
│   └── generate-jwt.sh   # JWT generation script
├── docs/                 # Documentation
├── tests/                # Test files
├── migrations/           # Database migrations
├── .env.example          # Environment template
├── go.mod                # Go modules
└── README.md             # This file
```

### Understanding Clean Architecture

ERPGo follows Clean Architecture principles:

1. **Domain Layer** (`internal/domain/`) - Core business rules
2. **Application Layer** (`internal/application/`) - Use cases and business logic
3. **Infrastructure Layer** (`internal/infrastructure/`) - External services
4. **Interface Layer** (`internal/interfaces/`) - API and user interfaces

## 🛠️ Development Guide

### Making Changes

1. **Always test your changes**
   ```bash
   go test ./...
   ```

2. **Format your code**
   ```bash
   go fmt ./...
   ```

3. **Check for errors**
   ```bash
   go vet ./...
   ```

### Adding New Features

1. **Start with the domain layer** - Define business entities
2. **Add repositories** - Define data access interfaces
3. **Implement services** - Write business logic
4. **Create handlers** - Add HTTP endpoints
5. **Write tests** - Ensure quality

### Common Development Commands

```bash
# Build the application
go build ./cmd/api

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Update dependencies
go mod tidy

# Get new dependency
go get package-name

# Generate API documentation
swag init
```

## 🐛 Troubleshooting

### Common Issues and Solutions

#### 1. "JWT_SECRET must be set to a secure value"

**Problem**: You haven't set a JWT secret or it's still the default value.

**Solution**:
```bash
# Generate a new secret
openssl rand -base64 32

# Edit your .env file
nano .env

# Replace the JWT_SECRET line with your new secret
JWT_SECRET=your-new-secret-here
```

#### 2. "Database connection failed"

**Problem**: PostgreSQL isn't running or connection details are wrong.

**Solution**:
```bash
# Check if PostgreSQL is running
brew services list | grep postgresql

# Start PostgreSQL if needed
brew services start postgresql

# Test connection manually
psql -h localhost -U erpgo -d erpgo

# Check your .env file settings
cat .env | grep DB_
```

#### 3. "Port 8080 is already in use"

**Problem**: Another application is using port 8080.

**Solution**:
```bash
# Find what's using the port
lsof -i :8080

# Kill the process (replace PID with actual process ID)
kill -9 PID

# Or change the port in .env
APP_PORT=8081
```

#### 4. "Go command not found"

**Problem**: Go isn't installed or not in your PATH.

**Solution**:
```bash
# Check if Go is installed
which go

# If not installed, download from golang.org
# Add Go to your PATH (add to ~/.zshrc or ~/.bash_profile)
export PATH=$PATH:/usr/local/go/bin

# Reload your shell
source ~/.zshrc  # or ~/.bash_profile
```

#### 5. "Permission denied" when running scripts

**Problem**: Script files don't have execute permissions.

**Solution**:
```bash
# Make the script executable
chmod +x scripts/generate-jwt.sh

# Try running again
./scripts/generate-jwt.sh
```

### Getting Help

1. **Check the logs** - Run the application with `LOG_LEVEL=debug`
2. **Search existing issues** - Check the project's issue tracker
3. **Ask for help** - Include:
   - Your operating system
   - Go version (`go version`)
   - The exact error message
   - What you were trying to do

## ❓ FAQ (Frequently Asked Questions)

### Q: What is an ERP system?
A: An ERP (Enterprise Resource Planning) system integrates various business processes into a single system. It helps manage day-to-day business activities like accounting, procurement, project management, risk management, and supply chain operations.

### Q: Why use Go for an ERP system?
A: Go offers excellent performance for concurrent operations, strong typing for reliability, and simple deployment. It's perfect for business applications that need to handle many users simultaneously while maintaining data integrity.

### Q: Do I need to know Go to use ERPGo?
A: For basic usage, no. You just need to understand how to make API requests. For customizations or development, yes, you'll need Go knowledge.

### Q: Can I use this without Docker?
A: Yes! The setup guide above shows how to run it natively. Docker is optional.

### Q: How secure is the JWT system?
A: The JWT system uses industry-standard practices with secure secrets, configurable expiration times, and proper validation. Always use strong secrets and HTTPS in production.

### Q: Can I add custom fields to users/products?
A: Yes, the system is designed to be extensible. You'll need to modify the domain entities, update the database schema, and adjust the API handlers.

### Q: How do I deploy this to production?
A: Key steps for production:
1. Use environment variables instead of .env file
2. Set strong secrets and passwords
3. Use HTTPS
4. Set up proper database backups
5. Configure logging and monitoring
6. Use reverse proxy (nginx/Apache)

### Q: Can I integrate with other systems?
A: Yes, the REST API allows integration with any system that can make HTTP requests. You can also extend the system with additional interfaces.

### Q: How do I reset the database?
A: ```bash
# Drop and recreate the database
dropdb erpgo
createdb erpgo
```

### Q: Where can I find more documentation?
A: Check the `docs/` directory for additional documentation on specific topics like JWT generation, API documentation, and deployment guides.

## 🎯 Next Steps

Now that you have ERPGo running:

1. **Explore the API** - Try different endpoints with your JWT token
2. **Read the API documentation** - Check `docs/API_DOCUMENTATION.md`
3. **Set up monitoring** - Configure Prometheus and Grafana
4. **Deploy to staging** - Try deploying to a staging environment
5. **Customize for your needs** - Add custom fields or features

## 📞 Support

If you encounter issues:

1. Check the troubleshooting section above
2. Look at the documentation in the `docs/` folder
3. Check the project's issue tracker
4. Create a new issue with detailed information

---

**Happy coding!** 🎉

You now have a complete ERP system running locally. The possibilities are endless for what you can build on top of this foundation.