"use client";

import React from "react";

export function LoadingState() {
  return (
    <main className="min-h-screen bg-background">
      <div className="container mx-auto px-4 py-8 md:py-16 max-w-4xl">
        <div className="animate-pulse">
          {/* Title skeleton */}
          <div className="h-12 bg-primary/20 rounded-lg w-64 mx-auto mb-8" />

          {/* Card skeleton */}
          <div className="p-8 shadow-lg border-2 rounded-lg space-y-6">
            {/* Password input skeleton */}
            <div className="space-y-2">
              <div className="h-4 bg-primary/20 rounded w-20" />
              <div className="h-10 bg-primary/10 rounded-md w-full" />
            </div>

            {/* Captcha skeleton */}
            <div className="h-24 bg-primary/10 rounded-md w-full" />

            {/* Button skeleton */}
            <div className="h-10 bg-primary/20 rounded-md w-full" />
          </div>
        </div>
      </div>
    </main>
  );
}
