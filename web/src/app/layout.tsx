import { Inter } from "next/font/google";
import "./globals.css";
import { defaultMetadata, viewport } from "./metadata";
import { Toaster } from "sonner";
import { Footer } from "@/components/footer";

const inter = Inter({ subsets: ["latin"] });

export const metadata = defaultMetadata;
export { viewport };

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <body
        className={`${inter.className} min-h-screen flex flex-col bg-black text-white`}
        suppressHydrationWarning
      >
        <main className="flex-1">{children}</main>
        <Footer />
        <Toaster position="top-right" />
      </body>
    </html>
  );
}
