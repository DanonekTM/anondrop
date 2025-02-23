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

The application is designed to be served behind nginx. Example configuration found in `nginx.conf`.

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
