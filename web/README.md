# Secrets Share Frontend

A Next.js frontend for the Secrets Share service, providing a modern and secure interface for sharing encrypted messages.

## Project Setup

1. **Environment Setup**:
   Create a `.env.local` file in the `web` directory:

   ```env
   # API Configuration
   NEXT_PUBLIC_API_URL=http://localhost:8081

   # Cloudflare Turnstile
   NEXT_PUBLIC_TURNSTILE_SITE_KEY=your_turnstile_site_key
   ```

2. **Install Dependencies**:
   ```bash
   npm install
   # or
   yarn install
   ```

## Development

Run the development server:

```bash
npm run dev
# or
yarn dev
```

The development server will be available at [http://localhost:3000](http://localhost:3000).

## Production Build

For production deployment (e.g., with nginx):

1. **Build the Application**:

   ```bash
   npm run build
   # or
   yarn build
   ```

2. **Output**:
   The build will create a static export in the `out` directory, which can be served by nginx.

## Nginx Configuration

The application is designed to be served behind nginx. Example configuration:

```nginx
server {
    listen 80;
    server_name your_domain.com;

    root /usr/share/nginx/html;
    index index.html;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN";
    add_header X-XSS-Protection "1; mode=block";
    add_header X-Content-Type-Options "nosniff";
    add_header Referrer-Policy "strict-origin-when-cross-origin";
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' challenges.cloudflare.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https:;";

    # Handle Next.js client-side routing
    location / {
        try_files $uri $uri.html $uri/ /index.html;
        add_header Cache-Control "no-store, no-cache, must-revalidate";
    }

    # Handle secret view routes
    location ~ ^/view/[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$ {
        try_files $uri $uri.html $uri/index.html /view/placeholder/index.html;
        add_header Cache-Control "no-store, no-cache, must-revalidate";
    }

    # Cache static files
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }
}
```

## Project Structure

```
web/
├── app/                  # Next.js app directory
│   ├── layout.tsx        # Root layout
│   ├── page.tsx          # Home page
│   └── view/             # Secret viewing pages
├── components/           # React components
├── lib/                  # Utility functions
├── public/               # Static assets
└── styles/               # CSS styles
```

## Features

- Modern, responsive UI
- Client-side encryption
- Cloudflare Turnstile integration
- Custom expiry time selection
- Burn after reading support
- Custom secret names

## Metadata

The application's metadata can be customized in `app/metadata.ts`:

```typescript
export const metadata = {
  title: "Secrets Share",
  description: "Secure secret sharing service",
  keywords: ["secret sharing", "encryption", "security"],
  authors: [{ name: "Your Name" }],
  // Add other metadata as needed
};
```

## Development Notes

- The application uses the App Router feature of Next.js 14
- Static export is used for production builds
- All encryption is performed client-side before sending to the API
- Environment variables prefixed with `NEXT_PUBLIC_` are exposed to the browser

## Learn More

- [Next.js Documentation](https://nextjs.org/docs)
- [Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/)
- [Web Crypto API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API)
