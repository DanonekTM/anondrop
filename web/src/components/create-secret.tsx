"use client";

import { useState, useRef, useEffect } from "react";
import { motion } from "framer-motion";
import { toast } from "sonner";
import { addMinutes } from "date-fns";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Eye, EyeOff, Info, Link, Copy, Check } from "lucide-react";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Turnstile } from "@/components/ui/turnstile";
import { createSecret } from "@/lib/api";
import { cn } from "@/lib/utils";
import { encrypt } from "@/lib/encryption";
import { Checkbox } from "@/components/ui/checkbox";

const EXPIRY_OPTIONS = [
  { value: "10m", label: "10 minutes", minutes: 10 },
  { value: "30m", label: "30 minutes", minutes: 30 },
  { value: "1h", label: "1 hour", minutes: 60 },
  { value: "24h", label: "1 day", minutes: 24 * 60 },
  { value: "7d", label: "7 days", minutes: 7 * 24 * 60 },
] as const;

export function CreateSecret() {
  const [message, setMessage] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [customName, setCustomName] = useState("");
  const [loading, setLoading] = useState(false);
  const [captchaToken, setCaptchaToken] = useState<string>();
  const [secretUrl, setSecretUrl] = useState<string>();
  const [expiryOption, setExpiryOption] = useState<string>("10m");
  const [isBurnAfterReading, setIsBurnAfterReading] = useState(false);
  const [secretInfo, setSecretInfo] = useState<{ customName?: string }>({});
  const captchaRef = useRef<HTMLDivElement>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [successDialogOpen, setSuccessDialogOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [activeTooltip, setActiveTooltip] = useState<string | null>(null);

  // Regex for URL-friendly characters (only alphanumeric)
  const NAME_REGEX = /^[a-zA-Z0-9]*$/;

  const handleCopy = async (textOrEvent: string | React.MouseEvent) => {
    try {
      let valueToCopy: string | undefined;

      if (typeof textOrEvent === "string") {
        valueToCopy = textOrEvent;
      } else {
        valueToCopy = secretUrl;
      }

      if (!valueToCopy) {
        toast.error("Nothing to copy");
        return;
      }

      await navigator.clipboard.writeText(valueToCopy);
      setCopied(true);
      toast.success("Copied to clipboard!");
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error("Failed to copy:", error);
      toast.error("Failed to copy to clipboard");
    }
  };

  const getExpiryDate = (option?: string) => {
    if (!option) return undefined;
    const expiry = EXPIRY_OPTIONS.find((opt) => opt.value === option);
    if (!expiry) return undefined;
    return addMinutes(new Date(), expiry.minutes);
  };

  const getSecretDescription = () => {
    if (isBurnAfterReading) {
      return "This secret can only be viewed once";
    }

    if (!expiryOption) return "";

    const expiry = EXPIRY_OPTIONS.find((opt) => opt.value === expiryOption);
    if (!expiry) return "";

    return `This secret will expire in ${expiry.label.toLowerCase()}`;
  };

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

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!message.trim()) {
      newErrors.secret = "Secret message is required";
    }

    if (!password) {
      newErrors.password = "Password is required";
    }

    if (customName && !NAME_REGEX.test(customName)) {
      newErrors.customName =
        "Custom name can only contain letters and numbers (A-Z, a-z, 0-9)";
    }

    if (customName && customName.length > 32) {
      newErrors.customName = "Custom name cannot exceed 32 characters";
    }

    if (!captchaToken) {
      newErrors.captcha = "Please complete the captcha verification";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (loading) return;

    // Clear previous errors
    setErrors({});

    // Validate form
    if (!validateForm()) {
      return;
    }

    setLoading(true);

    try {
      // Encrypt the message client-side
      const encryptedContent = await encrypt(message, password);

      // Create the secret
      const expiryDate = getExpiryDate(expiryOption);
      const response = await createSecret({
        encryptedContent,
        customName: customName || undefined,
        expiresAt: !isBurnAfterReading ? expiryDate?.toISOString() : undefined,
        maxViews: isBurnAfterReading ? 1 : undefined,
        captchaToken: captchaToken!,
      });

      // Generate the secret URL
      const newSecretUrl = `${window.location.origin}/view/?id=${response.id}`;
      setSecretUrl(newSecretUrl);
      setSecretInfo({ customName: customName || undefined });
      setSuccessDialogOpen(true);

      // Reset form
      setMessage("");
      setPassword("");
      setCustomName("");
      setCaptchaToken(undefined);
      resetCaptcha();
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Failed to create secret";

      // Handle specific error cases
      if (errorMessage.includes("name") && errorMessage.includes("taken")) {
        setErrors((prev) => ({
          ...prev,
          customName: "This custom name is already taken",
        }));
      } else if (errorMessage.includes("password")) {
        setErrors((prev) => ({ ...prev, password: errorMessage }));
      } else if (errorMessage.includes("captcha")) {
        setErrors((prev) => ({
          ...prev,
          captcha: "Invalid captcha verification",
        }));
        resetCaptcha();
      } else {
        toast.error(errorMessage);
        resetCaptcha();
      }
    } finally {
      setLoading(false);
    }
  };

  const renderInfo = (content: string) => {
    const tooltipId = content.slice(0, 20); // Use part of content as unique ID

    return (
      <TooltipProvider>
        <Tooltip open={activeTooltip === tooltipId}>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-4 w-4 p-0 ml-2"
              onClick={(e) => {
                e.preventDefault();
                setActiveTooltip(
                  activeTooltip === tooltipId ? null : tooltipId
                );
              }}
            >
              <Info className="h-4 w-4 text-muted-foreground hover:text-foreground" />
              <span className="sr-only">More information</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent
            side="top"
            className="max-w-[280px] text-sm break-words bg-popover border border-border shadow-md px-3 py-2 rounded-md mx-2 w-[calc(100vw-1rem)] sm:w-auto"
            sideOffset={5}
          >
            {content}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  };

  // Add click outside handler to close tooltip
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        activeTooltip &&
        !(event.target as Element).closest('[role="tooltip"]')
      ) {
        setActiveTooltip(null);
      }
    };

    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [activeTooltip]);

  return (
    <>
      <Card className="p-8 shadow-lg border-2">
        <motion.form
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -20 }}
          onSubmit={handleSubmit}
          className="space-y-6"
        >
          <div className="space-y-2">
            <Label htmlFor="message">Secret Message</Label>
            <Textarea
              id="message"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Enter your secret message"
              className={cn(
                "min-h-[100px]",
                errors.secret &&
                  "border-destructive focus-visible:ring-destructive"
              )}
              required
            />
            {errors.secret && (
              <p className="text-sm text-destructive">{errors.secret}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? "text" : "password"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter a secure password"
                className={cn(
                  errors.password &&
                    "border-destructive focus-visible:ring-destructive"
                )}
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
            {errors.password && (
              <p className="text-sm text-destructive">{errors.password}</p>
            )}
          </div>

          <div className="space-y-2">
            <div className="flex items-center">
              <Label htmlFor="customName">Custom Name (optional)</Label>
              {renderInfo(
                "Optional but if you don't set it you will need to use the link to view the secret. Only letters and numbers are allowed."
              )}
            </div>
            <Input
              id="customName"
              value={customName}
              onChange={(e) => {
                const newValue = e.target.value;
                if (newValue === "" || NAME_REGEX.test(newValue)) {
                  setCustomName(newValue);
                  setErrors((prev) => ({ ...prev, customName: "" }));
                }
              }}
              placeholder="Enter a memorable name for your secret (letters and numbers only)"
              className={cn(
                errors.customName &&
                  "border-destructive focus-visible:ring-destructive"
              )}
              autoComplete="off"
              autoCorrect="off"
              spellCheck="false"
              pattern="[a-zA-Z0-9]*"
              title="Only letters and numbers are allowed"
              maxLength={32}
            />
            {errors.customName && (
              <p className="text-sm text-destructive">{errors.customName}</p>
            )}
          </div>

          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="burn"
                checked={isBurnAfterReading}
                onCheckedChange={(checked: boolean) => {
                  setIsBurnAfterReading(checked);
                  if (checked) {
                    setExpiryOption("");
                  } else {
                    setExpiryOption("10m");
                  }
                }}
              />
              <div className="flex items-center">
                <Label htmlFor="burn" className="cursor-pointer">
                  Burn after reading
                </Label>
                {renderInfo(
                  "The secret will be permanently deleted after being viewed once."
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label
                className={cn(isBurnAfterReading && "text-muted-foreground")}
              >
                Expiry Time
                {renderInfo(
                  isBurnAfterReading
                    ? "Expiry time is not applicable for burn-after-reading secrets"
                    : "Select when the secret should expire"
                )}
              </Label>
              <RadioGroup
                value={expiryOption}
                disabled={isBurnAfterReading}
                onValueChange={setExpiryOption}
                className="grid gap-4"
              >
                {EXPIRY_OPTIONS.map((option) => (
                  <div
                    key={option.value}
                    className="flex items-center space-x-2"
                  >
                    <RadioGroupItem
                      value={option.value}
                      id={option.value}
                      disabled={isBurnAfterReading}
                    />
                    <Label
                      htmlFor={option.value}
                      className={cn(
                        "cursor-pointer",
                        isBurnAfterReading && "text-muted-foreground"
                      )}
                    >
                      {option.label}
                    </Label>
                  </div>
                ))}
              </RadioGroup>
            </div>
          </div>

          <Turnstile
            onVerify={setCaptchaToken}
            id="create-secret"
            ref={captchaRef}
          />
          {errors.captcha && (
            <p className="text-sm text-destructive">{errors.captcha}</p>
          )}

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "Creating..." : "Create Secret"}
          </Button>
        </motion.form>
      </Card>

      <Dialog open={successDialogOpen} onOpenChange={setSuccessDialogOpen}>
        <DialogContent className="w-[calc(100%-2rem)] sm:max-w-lg mx-auto">
          <DialogHeader className="!text-left">
            <DialogTitle>Secret Created Successfully</DialogTitle>
            <DialogDescription className="space-y-6 pt-4 text-left">
              <div className="text-sm space-y-4">
                <div>{getSecretDescription()}</div>
                <div>After that, it will be permanently deleted.</div>
              </div>
              {secretInfo.customName && (
                <div className="text-sm">
                  <div className="flex items-center gap-2">
                    <span>You can also view this secret using this name:</span>
                    <code className="relative rounded bg-muted px-[0.5rem] py-[0.2rem] font-mono text-sm font-semibold">
                      {secretInfo.customName}
                    </code>
                  </div>
                </div>
              )}
              <div className="border-t pt-4">
                <div className="flex items-center space-x-2 bg-muted p-3 rounded-lg overflow-hidden max-w-full">
                  <Link className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                  <Input
                    value={secretUrl}
                    readOnly
                    className="bg-transparent border-none focus:ring-0 focus:ring-offset-0 focus-visible:ring-0 focus-visible:ring-offset-0 focus:border-none focus-visible:border-none overflow-x-auto whitespace-nowrap text-ellipsis cursor-text min-w-0 flex-1"
                    style={{ scrollbarWidth: "thin" }}
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleCopy}
                    className="flex-shrink-0"
                  >
                    {copied ? (
                      <Check className="h-4 w-4" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                    <span className="sr-only">Copy link</span>
                  </Button>
                </div>
                <p className="text-sm text-muted-foreground mt-2">
                  Share this link with the recipient. They will need the
                  password you set to view the secret.
                </p>
              </div>
            </DialogDescription>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    </>
  );
}
