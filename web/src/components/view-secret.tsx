"use client";

import { useState, useRef } from "react";
import { motion } from "framer-motion";
import { toast } from "sonner";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Turnstile } from "@/components/ui/turnstile";
import { viewSecretByName } from "@/lib/api";
import { formatDistanceToNow } from "date-fns";
import { Eye, EyeOff } from "lucide-react";
import { decrypt } from "@/lib/encryption";
import { useContext } from "react";
import { TabsContext } from "@/app/tabs-context";
import { cn } from "@/lib/utils";

export function ViewSecret() {
  const [name, setName] = useState("");
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
  const [error, setError] = useState("");

  // Regex for URL-friendly characters (only alphanumeric)
  const NAME_REGEX = /^[a-zA-Z0-9]*$/;

  // Store the encrypted content locally for secrets, keyed by name
  const [encryptedContentMap, setEncryptedContentMap] = useState<
    Record<
      string,
      {
        encrypted: string;
        salt: string;
        iv: string;
        isBurnAfterReading: boolean;
        expiresAt?: string;
      }
    >
  >({});

  const { setActiveTab } = useContext(TabsContext);

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

  const handleDecryption = async (
    encryptedContent: { encrypted: string; salt: string; iv: string },
    password: string
  ) => {
    try {
      const decryptedMessage = await decrypt(
        encryptedContent.encrypted,
        encryptedContent.salt,
        encryptedContent.iv,
        password
      );
      setMessage(decryptedMessage);
      return true;
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
      return false;
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (loading) return;

    // Validate name format
    if (!NAME_REGEX.test(name)) {
      toast.error(
        "Secret name can only contain letters and numbers (A-Z, a-z, 0-9)"
      );
      return;
    }

    setLoading(true);

    try {
      // Check if we already have the encrypted content for this name
      const storedContent = encryptedContentMap[name];

      if (storedContent) {
        // Try to decrypt with stored content
        const success = await handleDecryption(storedContent, password);
        if (success) {
          setSecretInfo({
            expiresAt: storedContent.expiresAt,
            isBurnAfterReading: storedContent.isBurnAfterReading,
          });
        } else {
          setPassword("");
        }
        setLoading(false);
        return;
      }

      // If we don't have stored content, we need to fetch from the backend
      if (!captchaToken) {
        toast.error("Please complete the captcha verification");
        setLoading(false);
        return;
      }

      const data = await viewSecretByName(name, { captchaToken });

      // Store the encrypted content in our map
      setEncryptedContentMap((prev) => ({
        ...prev,
        [name]: {
          ...data.encryptedContent,
          isBurnAfterReading: data.maxViews === 1,
          expiresAt: data.expiresAt,
        },
      }));

      const success = await handleDecryption(data.encryptedContent, password);
      if (success) {
        setSecretInfo({
          expiresAt: data.expiresAt,
          isBurnAfterReading: data.maxViews === 1,
        });
        setPassword("");
        setCaptchaToken(undefined);
        resetCaptcha();
      } else {
        setPassword("");
        resetCaptcha();
        setCaptchaToken(undefined);
      }
    } catch (error) {
      let errorMessage =
        error instanceof Error ? error.message : "Failed to view secret";
      if (errorMessage.includes("not found")) {
        errorMessage = "Secret not found. Please check the name and try again.";
      }
      toast.error(errorMessage);
      resetCaptcha();
      setCaptchaToken(undefined);
    } finally {
      setLoading(false);
    }
  };

  return (
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
            <Label htmlFor="name">Secret Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => {
                const newValue = e.target.value;
                if (newValue === "" || NAME_REGEX.test(newValue)) {
                  setName(newValue);
                  setError("");
                }
              }}
              placeholder="Enter the secret's name (letters and numbers only)"
              required
              autoComplete="off"
              autoCorrect="off"
              spellCheck="false"
              pattern="[a-zA-Z0-9]*"
              title="Only letters and numbers are allowed"
              maxLength={32}
              className={cn(
                error && "border-destructive focus-visible:ring-destructive"
              )}
            />
            {error && <p className="text-sm text-destructive mt-1">{error}</p>}
          </div>

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
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                onClick={() => setShowPassword(!showPassword)}
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

          <div>
            {!encryptedContentMap[name] && (
              <Turnstile
                onVerify={setCaptchaToken}
                id="view-secret"
                ref={captchaRef}
              />
            )}
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "Loading..." : "View Secret"}
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
              setActiveTab("create");
              setMessage("");
              setPassword("");
              setName("");
              setCaptchaToken(undefined);
              setSecretInfo(undefined);
              setEncryptedContentMap({});
            }}
            className="w-full"
          >
            Create Another Secret
          </Button>
        </motion.div>
      )}
    </Card>
  );
}
