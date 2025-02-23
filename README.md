<div align="center">
  <img src="/images/logo.png" alt="AnonDrop Logo" width="200">
</div>

# AnonDrop.link

A secure secret sharing service that allows users to share encrypted messages with expiry options and custom names.

## Screenshots

### Frontend Interface

![AnonDrop Frontend](/images/anondrop-home.png)
_The main interface where users can create and share encrypted secrets_

### Server Logs

![AnonDrop Server](/images/anondrop-server.png)
_Server logs showing the application's runtime information and request handling_

## Features

- Create encrypted secrets with passwords
- Optional custom names for secrets
- Time-based expiry (10 minutes, 30 minutes, 1 hour, 1 day, or 7 days)
- Burn after reading (single view) functionality
- Cloudflare Turnstile captcha protection
- Client-side and server-side encryption
- Modern Next.js frontend
- REST API with JSON endpoints
- Optional rate limiting with Redis
- File-based storage for secrets

## Project Structure

```
.
├── cmd/                   # Application entry points
├── internal/              # Internal application code
├── web/                   # Next.js Frontend application
├── data/                  # Data storage
│   └── secrets/           # Encrypted secrets storage
└── logs/                  # Application logs
```

## Requirements

- Go 1.24 or later
- Node.js v23.6.1 or later
- Redis (optional, for rate limiting)

## Development Setup

### Local Development (Direct)

1. **Backend Setup**:

   ```bash
   # Install Go dependencies
   go mod download

   # Create necessary directories
   mkdir -p data/secrets logs

   # Start Redis (optional, only needed if you want rate limiting)
   redis-server

   # Run the backend
   go run cmd/main.go
   ```

2. **Frontend Setup**:

   ```bash
   # Navigate to frontend directory
   cd web

   # Install dependencies
   npm install

   # Start development server
   npm run dev
   ```

### Running Tests

1. **Backend Tests**:

   ```bash
   # Run all Go tests with verbose output
   go test ./... -v

   # Run tests for a specific package
   go test ./internal/encryption -v
   go test ./internal/api/handlers -v
   ```

2. **Frontend Tests**:

   ```bash
   # Navigate to frontend directory
   cd web

   # Run Jest tests
   npm run test

   # Run tests in watch mode
   npm run test -- --watch
   ```

## Configuration

### Environment Variables (.env)

Create a `.env` file in the root directory:

```env
# Server Encryption (Required)
SERVER_ENCRYPTION_KEY=your-secure-encryption-key

# Redis Configuration (Optional)
REDIS_PASSWORD=your-redis-password
REDIS_USERNAME=your-redis-username

# Cloudflare Turnstile (Required)
CAPTCHA_SECRET_KEY=your-captcha-secret
```

### Application Configuration (config.yaml)

The `config.yaml` file contains application settings including:

- Server configuration
- Security settings
- Rate limiting rules (when Redis is enabled)
- Secret storage settings
- Logging configuration

### Frontend Configuration

The frontend configuration is detailed in the [web/README.md](web/README.md) file, which includes:

- Environment setup (.env.local)
- Development instructions
- Production build steps
- Nginx configuration
- Project structure
- Features overview
- Development notes

## Production Deployment

Currently, the only way to deploy this is manually, I'm working on a containerized solution/kubernetes deployment.

**Manual Deployment**:

```bash
# Build the backend
go build -o anondrop cmd/main.go

# Create systemd service
sudo nano /etc/systemd/system/anondrop.service

[Unit]
Description=AnonDrop Secret Sharing Service
After=network.target

[Service]
Type=simple
User=anondrop
WorkingDirectory=/path/to/anondrop
ExecStart=/path/to/anondrop/executable
Restart=always

[Install]
WantedBy=multi-user.target

# Start service
sudo systemctl enable anondrop
sudo systemctl start anondrop

# Build the frontend
cd web
npm run build

# Copy static files and nginx config
sudo mkdir -p /path/to/anondrop/public
sudo cp -r out/* /path/to/anondrop/public
sudo cp nginx.conf /etc/nginx/conf.d/anondrop.conf

# Restart nginx
sudo systemctl restart nginx
```

## API Endpoints

The REST API is available at `/api`. Main endpoints:

1. **Create a secret**:

   ```http
   POST /api/secrets
   Content-Type: application/json

   {
     "encryptedContent": {
       "encrypted": "base64_encrypted_data",
       "salt": "base64_salt",
       "iv": "base64_iv"
     },
     "customName": "optional_name",
     "expiresAt": "2024-02-23T15:00:00Z",
     "maxViews": 1,
     "captchaToken": "turnstile_token"
   }
   ```

2. **View a secret**:

   ```http
   POST /api/secrets/{id}
   Content-Type: application/json

   {
     "captchaToken": "turnstile_token"
   }
   ```

3. **View a secret by custom name**:

   ```http
   POST /api/secrets/name/{name}
   Content-Type: application/json

   {
     "captchaToken": "turnstile_token"
   }
   ```

## Security Considerations

- All secrets are encrypted using AES-256-GCM
- Client-side encryption with unique salt and IV per secret
- Additional server-side encryption layer
- Cloudflare Turnstile protection against bots
- Optional rate limiting with Redis
- Automatic cleanup of expired secrets
- CORS protection
- Maximum secret size limit

## License

MIT License
