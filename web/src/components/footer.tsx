"use client";

import { useEffect, useState } from "react";
import { InfoButton } from "./info-button";

export function Footer() {
  const [year, setYear] = useState(new Date().getFullYear());

  useEffect(() => {
    // Update year when component mounts and at midnight
    const interval = setInterval(() => {
      setYear(new Date().getFullYear());
    }, 1000 * 60 * 60); // Check every hour

    return () => clearInterval(interval);
  }, []);

  return (
    <>
      {/* Mobile Footer - Appears at bottom of content */}
      <footer className="md:hidden mt-auto py-6 bg-black">
        <div className="container max-w-4xl mx-auto flex items-center justify-between px-4">
          <InfoButton />
          <div className="text-sm text-muted-foreground">
            © {year} anondrop.link
          </div>
        </div>
      </footer>

      {/* Desktop Fixed Elements */}
      <div className="hidden md:block">
        <div className="fixed bottom-4 left-4">
          <InfoButton />
        </div>
        <div className="fixed bottom-4 right-4 text-sm text-muted-foreground">
          © {year} anondrop.link
        </div>
      </div>
    </>
  );
}
