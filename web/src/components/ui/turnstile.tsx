"use client";

import { useEffect, useRef, useCallback } from "react";
import { useTheme } from "next-themes";
import React from "react";

declare global {
  interface TurnstileOptions {
    sitekey: string;
    callback: (token: string) => void;
    theme?: "light" | "dark" | "auto";
    appearance?: "always" | "execute" | "interaction-only";
    size?: "normal" | "compact";
    retry?: "auto" | "never";
    "refresh-expired"?: "auto" | "manual" | "never";
    language?: string;
    "retry-if-not-visible"?: boolean;
  }

  interface Window {
    turnstile: {
      render: (
        container: string | HTMLElement,
        options: TurnstileOptions
      ) => string;
      reset: (widgetId: string) => void;
      remove: (widgetId: string) => void;
    };
    onTurnstileLoad: () => void;
  }
}

interface TurnstileProps {
  onVerify: (token: string) => void;
  id: string;
}

const Turnstile = React.forwardRef<HTMLDivElement, TurnstileProps>(
  ({ onVerify, id }, ref) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const widgetIdRef = useRef<string | undefined>(undefined);
    const { theme } = useTheme();

    const handleVerify = useCallback(
      (token: string) => {
        onVerify(token);
      },
      [onVerify]
    );

    const reset = () => {
      if (widgetIdRef.current && window.turnstile) {
        window.turnstile.reset(widgetIdRef.current);
      }
    };

    useEffect(() => {
      const currentContainer = containerRef.current;
      if (!currentContainer) return;

      // Load the Turnstile script only once
      if (!document.querySelector('script[src*="turnstile"]')) {
        const script = document.createElement("script");
        script.src =
          "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit&onload=onTurnstileLoad";
        script.async = true;
        script.defer = true;
        document.body.appendChild(script);
      }

      const renderWidget = () => {
        if (!currentContainer || !window.turnstile) return;

        // Only create a new widget if one doesn't exist
        if (!widgetIdRef.current) {
          widgetIdRef.current = window.turnstile.render(currentContainer, {
            sitekey: process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY as string,
            theme: theme === "dark" ? "dark" : "light",
            callback: handleVerify,
            "refresh-expired": "manual",
            "retry-if-not-visible": false,
            appearance: "interaction-only",
          });
        }
      };

      // Define the global callback function
      window.onTurnstileLoad = renderWidget;

      // If turnstile is already loaded, call renderWidget directly
      if (window.turnstile) {
        renderWidget();
      }

      return () => {
        if (widgetIdRef.current && window.turnstile) {
          try {
            window.turnstile.remove(widgetIdRef.current);
            widgetIdRef.current = undefined;
          } catch (e) {
            console.error("Failed to remove widget during cleanup:", e);
          }
        }
      };
    }, [handleVerify, theme, id]);

    return (
      <div
        id={`turnstile-${id}`}
        ref={(element) => {
          // Update both refs
          containerRef.current = element;
          if (typeof ref === "function") {
            ref(element);
          } else if (ref) {
            ref.current = element;
          }
        }}
        className="mt-4"
        data-widget-id={widgetIdRef.current}
        data-reset-function={reset.name}
      />
    );
  }
);

Turnstile.displayName = "Turnstile";

export { Turnstile };
