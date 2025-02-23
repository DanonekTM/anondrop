import { Metadata, Viewport } from 'next';

// Base URL configuration for metadata
const BASE_URL = process.env.NEXT_PUBLIC_BASE_URL || 'https://anondrop.link';

// Default metadata configuration
export const defaultMetadata: Metadata = {
  metadataBase: new URL(BASE_URL),
  title: 'AnonDrop.link - Secure Secret Sharing',
  description: 'Drop your secret. A one-time message, then it\'s gone.',
  icons: {
    icon: [
      { url: '/icons/favicon-16x16.png', sizes: '16x16', type: 'image/png' },
      { url: '/icons/favicon-32x32.png', sizes: '32x32', type: 'image/png' },
      { url: '/icons/favicon-48x48.png', sizes: '48x48', type: 'image/png' },
      { url: '/icons/favicon-64x64.png', sizes: '64x64', type: 'image/png' },
      { url: '/icons/favicon-128x128.png', sizes: '128x128', type: 'image/png' },
      { url: '/icons/android-chrome-192x192.png', sizes: '192x192', type: 'image/png' },
      { url: '/icons/android-chrome-512x512.png', sizes: '512x512', type: 'image/png' },
    ],
    apple: [
      { url: '/icons/apple-touch-icon.png', sizes: '180x180', type: 'image/png' },
    ],
    shortcut: [{ url: '/icons/favicon.ico' }],
  },
  manifest: '/manifest.json',
  appleWebApp: {
    capable: true,
    statusBarStyle: 'default',
    title: 'AnonDrop.link',
  },
  openGraph: {
    type: 'website',
    siteName: 'AnonDrop.link',
    title: {
      default: 'AnonDrop.link - Secure Secret Sharing',
      template: '%s | AnonDrop.link'
    },
    description: 'Drop your secret. A one-time message, then it\'s gone.',
    url: BASE_URL,
    images: [{ 
      url: '/images/og-image.jpg',
      width: 1200,
      height: 630,
      alt: 'AnonDrop.link - Secure Secret Sharing'
    }],
  },
  twitter: {
    card: 'summary_large_image',
    title: {
      default: 'AnonDrop.link - Secure Secret Sharing',
      template: '%s | AnonDrop.link'
    },
    description: 'Drop your secret. A one-time message, then it\'s gone.',
    images: ['/images/twitter-image.jpg'],
  },
  applicationName: 'AnonDrop.link',
  referrer: 'origin-when-cross-origin',
  keywords: ['secret sharing', 'secure messaging', 'one-time message', 'encrypted sharing'],
  authors: [{ name: 'AnonDrop.link' }],
  creator: 'AnonDrop.link',
  publisher: 'AnonDrop.link',
};

// View page metadata
export const viewMetadata: Metadata = {
  title: 'AnonDrop.link - View Secret',
  description: 'Securely view your one-time secret message.',
  openGraph: {
    title: 'View Secret',
    description: 'Securely view your one-time secret message.',
    images: [{ 
      url: '/images/og-image.jpg',
      width: 1200,
      height: 630,
      alt: 'View Secret on AnonDrop.link'
    }],
  },
  twitter: {
    title: 'View Secret',
    description: 'Securely view your one-time secret message.',
  },
};

// 404 page metadata
export const notFoundMetadata: Metadata = {
  title: '404 - Page Not Found',
  description: 'The page you are looking for could not be found.',
  robots: {
    index: false,
    follow: true
  },
  openGraph: {
    title: '404 - Page Not Found',
    description: 'The page you are looking for could not be found.',
  },
};

// Viewport configuration
export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#ffffff' },
    { media: '(prefers-color-scheme: dark)', color: '#000000' },
  ],
}; 