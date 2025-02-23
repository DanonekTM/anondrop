"use client";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { HelpCircle } from "lucide-react";
import { useState } from "react";

export function InfoButton() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button variant="ghost" size="icon" onClick={() => setOpen(true)}>
        <HelpCircle className="h-5 w-5" />
        <span className="sr-only">Information</span>
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="w-[calc(100%-2rem)] sm:max-w-[425px] mx-auto">
          <DialogHeader className="!text-left">
            <DialogTitle>About AnonDrop</DialogTitle>
            <DialogDescription className="space-y-4 pt-4 overflow-x-hidden">
              <p className="text-left">
                AnonDrop.link lets you share confidential information privately
                and securely.
              </p>
              <div className="space-y-2">
                <h3 className="font-semibold text-left">Features:</h3>
                <ul className="list-disc pl-6 space-y-1 text-left">
                  <li>
                    End-to-end encryption ensures your secrets remain private
                  </li>
                  <li>Set expiry times for automatic deletion</li>
                  <li>Burn-after-reading option for one-time viewing</li>
                  <li>Custom names for easier secret retrieval</li>
                  <li>No registration required</li>
                  <li>No logs or tracking</li>
                </ul>
              </div>
              <div className="space-y-2">
                <h3 className="font-semibold text-left">How to use:</h3>
                <ol className="list-decimal pl-6 space-y-1 text-left">
                  <li>Enter your secret message</li>
                  <li>Set a strong password</li>
                  <li>Choose expiry time or burn-after-reading</li>
                  <li>
                    Share the generated link or the secret&apos;s name and
                    password with the recipient.
                  </li>
                </ol>
              </div>
              <p className="text-sm text-muted-foreground text-left">
                All secrets are automatically deleted after expiry or burned
                after reading.
              </p>
            </DialogDescription>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    </>
  );
}
