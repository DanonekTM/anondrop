"use client";

import { useState, useRef, Suspense, useEffect } from "react";
import { motion } from "framer-motion";
import { toast } from "sonner";
import { useRouter, useSearchParams } from "next/navigation";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Turnstile } from "@/components/ui/turnstile";
import { Eye, EyeOff, AlertTriangle } from "lucide-react";
import { Spinner } from "@/components/ui/spinner";
import Link from "next/link";
import { viewSecret } from "@/lib/api";
import { formatDistanceToNow } from "date-fns";
import { decrypt } from "@/lib/encryption";
import { LoadingState } from "@/components/loading-state";

const UUID_REGEX =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default function ViewPage() {
  return (
    <Suspense fallback={<LoadingState />}>
      <ViewSecretContent />
    </Suspense>
  );
}

function ViewSecretContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const id = searchParams.get("id") || "";

  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const [captchaToken, setCaptchaToken] = useState<string>();
  const captchaRef = useRef<HTMLDivElement>(null);
  const [secretInfo, setSecretInfo] = useState<{
    expiresAt?: string;
    isBurnAfterReading?: boolean;
  }>();
  const [encryptedContent, setEncryptedContent] = useState<{
    encrypted: string;
    salt: string;
    iv: string;
  } | null>(null);
  const [hasFetchedSecret, setHasFetchedSecret] = useState(false);

  // Clear secret data when window is closed/refreshed
  useEffect(() => {
    const handleBeforeUnload = () => {
      setEncryptedContent(null);
      setSecretInfo(undefined);
      setMessage("");
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, []);

  // If the ID is invalid, show the error state
  if (!id || !UUID_REGEX.test(id)) {
    return (
      <main className="min-h-screen bg-background flex items-center justify-center">
        <div className="container mx-auto px-4 max-w-md">
          <div className="text-center space-y-6">
            <h1 className="text-4xl md:text-5xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-primary to-primary/60">
              Invalid Secret
            </h1>
            <p className="text-lg text-muted-foreground">
              This secret ID is invalid or has been deleted.
            </p>
            <Button
              asChild
              className="w-full md:w-auto min-w-[200px]"
              size="lg"
            >
              <Link href="/">Create a new secret</Link>
            </Button>
          </div>
        </div>
      </main>
    );
  }

  const resetCaptcha = () => {
    const captchaElement = captchaRef.current;
    if (captchaElement) {
      const resetFunction = captchaElement.getAttribute("data-reset-function");
      if (resetFunction && window.turnstile) {
        const widgetId = captchaElement.getAttribute("data-widget-id");
        if (widgetId) {
          window.turnstile.reset(widgetId);
        }
      }
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      // If we haven't fetched the secret yet, fetch it from the API
      if (!hasFetchedSecret) {
        if (!captchaToken) {
          toast.error("Please complete the captcha verification");
          setLoading(false);
          return;
        }

        const response = await viewSecret(id, { captchaToken });

        if (response.error) {
          toast.error(response.error);
          resetCaptcha();
          setCaptchaToken(undefined);
          setLoading(false);
          return;
        }

        setEncryptedContent(response.encryptedContent);
        setSecretInfo({
          expiresAt: response.expiresAt,
          isBurnAfterReading: response.isBurnAfterReading,
        });
        setHasFetchedSecret(true);
        resetCaptcha();
        setCaptchaToken(undefined);

        // Try to decrypt with the initial password
        try {
          const decryptedMessage = await decrypt(
            response.encryptedContent.encrypted,
            response.encryptedContent.salt,
            response.encryptedContent.iv,
            password
          );
          setMessage(decryptedMessage);
        } catch (decryptError) {
          if (
            decryptError instanceof Error &&
            decryptError.name === "OperationError"
          ) {
            toast.error("Invalid password. Please try again.");
          } else {
            console.error("Unexpected decryption error:", decryptError);
            toast.error("Failed to decrypt the message. Please try again.");
          }
          setPassword("");
          setLoading(false);
          return;
        }
      } else {
        // For subsequent attempts, use the stored encrypted content
        if (!encryptedContent) {
          toast.error("No encrypted content available");
          setLoading(false);
          return;
        }

        try {
          const decryptedMessage = await decrypt(
            encryptedContent.encrypted,
            encryptedContent.salt,
            encryptedContent.iv,
            password
          );
          setMessage(decryptedMessage);
        } catch (decryptError) {
          if (
            decryptError instanceof Error &&
            decryptError.name === "OperationError"
          ) {
            toast.error("Invalid password. Please try again.");
          } else {
            console.error("Unexpected decryption error:", decryptError);
            toast.error("Failed to decrypt the message. Please try again.");
          }
          setPassword("");
          setLoading(false);
          return;
        }
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : "Failed to view secret";
      toast.error(
        errorMessage.includes("not found")
          ? "Secret not found or has been deleted"
          : errorMessage
      );
      resetCaptcha();
      setCaptchaToken(undefined);
      setHasFetchedSecret(false);
    } finally {
      setLoading(false);
    }
  };

  const getSecretStatusMessage = () => {
    if (!secretInfo) {
      return "This message is only visible until you close this tab";
    }

    if (secretInfo.isBurnAfterReading) {
      return "This message is only visible until you close this tab";
    }

    if (secretInfo.expiresAt) {
      const expiryDate = new Date(secretInfo.expiresAt);
      const timeLeft = formatDistanceToNow(expiryDate, { addSuffix: true });
      return `This secret will expire ${timeLeft}`;
    }

    return "This message is only visible until you close this tab";
  };

  return (
    <main className="min-h-screen bg-background">
      <div className="container mx-auto px-4 py-8 md:py-16 max-w-4xl">
        <h1 className="text-4xl md:text-5xl font-bold mb-8 text-center bg-clip-text text-transparent bg-gradient-to-r from-primary to-primary/60">
          View Secret
        </h1>
        <Card className="p-8 shadow-lg border-2">
          {!message ? (
            <motion.form
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              onSubmit={handleSubmit}
              className="space-y-6"
            >
              <div className="space-y-2">
                <Label htmlFor="viewPassword">Password</Label>
                <div className="relative">
                  <Input
                    id="viewPassword"
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Enter the secret's password"
                    required
                    disabled={loading}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                    onClick={() => setShowPassword(!showPassword)}
                    disabled={loading}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <Eye className="h-4 w-4 text-muted-foreground" />
                    )}
                    <span className="sr-only">
                      {showPassword ? "Hide password" : "Show password"}
                    </span>
                  </Button>
                </div>
              </div>

              {!hasFetchedSecret && (
                <div>
                  <Turnstile
                    onVerify={setCaptchaToken}
                    id="view-secret-by-id"
                    ref={captchaRef}
                  />
                </div>
              )}

              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? (
                  <div className="flex items-center justify-center gap-2">
                    <Spinner size="sm" />
                    <span>
                      {hasFetchedSecret
                        ? "Decrypting..."
                        : "Fetching secret..."}
                    </span>
                  </div>
                ) : (
                  "View Secret"
                )}
              </Button>
            </motion.form>
          ) : (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              className="space-y-4"
            >
              <h2 className="text-xl font-semibold">Secret Message</h2>
              <div className="bg-muted p-4 rounded-md">
                <pre className="whitespace-pre-wrap break-words">{message}</pre>
              </div>
              <p className="text-sm text-muted-foreground">
                {getSecretStatusMessage()}
              </p>
              <Button
                onClick={() => {
                  router.push("/");
                }}
                className="w-full"
              >
                Create Another Secret
              </Button>
            </motion.div>
          )}
        </Card>
        {secretInfo?.isBurnAfterReading && !message && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-4 p-4 bg-destructive/10 text-destructive rounded-lg"
          >
            <div className="flex items-center justify-center gap-2">
              <AlertTriangle className="h-5 w-5" />
              <p className="font-semibold">Warning: One-Time View Secret</p>
            </div>
            <p className="text-sm mt-1 text-center">
              This is a one-time view secret. If you leave or refresh this page,
              the secret will be permanently deleted and cannot be recovered.
            </p>
          </motion.div>
        )}
      </div>
    </main>
  );
}
