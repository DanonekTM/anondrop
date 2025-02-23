import Link from "next/link";
import { Home } from "lucide-react";
import { notFoundMetadata } from "./metadata";

export const metadata = notFoundMetadata;

export default function NotFound() {
  return (
    <div className="flex items-center justify-center min-h-screen w-full">
      <div className="flex flex-col items-center text-center px-4">
        <h2 className="text-4xl font-bold mb-4">404</h2>
        <p className="text-xl mb-8 text-gray-400">Page not found</p>
        <Link
          href="/"
          className="inline-flex items-center px-6 py-3 bg-white text-black rounded-lg hover:bg-gray-100 transition-colors duration-200 font-medium"
        >
          <Home className="w-5 h-5 mr-2" />
          Go back home
        </Link>
      </div>
    </div>
  );
}
