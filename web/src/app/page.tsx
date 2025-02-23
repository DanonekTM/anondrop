"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CreateSecret } from "@/components/create-secret";
import { ViewSecret } from "@/components/view-secret";
import { motion } from "framer-motion";
import { useState } from "react";
import { TabsContext } from "./tabs-context";

export default function Home() {
  const [activeTab, setActiveTab] = useState("create");

  return (
    <TabsContext.Provider value={{ activeTab, setActiveTab }}>
      <main className="min-h-screen bg-background">
        <div className="container mx-auto px-4 py-8 md:py-16 max-w-4xl">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="flex flex-col items-center justify-center"
          >
            <h1 className="text-4xl md:text-5xl font-bold mb-4 text-center bg-clip-text text-transparent bg-gradient-to-r from-primary to-primary/60">
              AnonDrop.link
            </h1>
            <p className="text-lg md:text-xl mb-8 text-center text-muted-foreground">
              Drop your secret. A one-time message, then it&apos;s gone.
            </p>
            <Tabs
              value={activeTab}
              onValueChange={setActiveTab}
              className="w-full"
            >
              <TabsList className="grid w-full grid-cols-2 mb-8">
                <TabsTrigger value="create">Create Secret</TabsTrigger>
                <TabsTrigger value="view">View Secret</TabsTrigger>
              </TabsList>
              <TabsContent value="create" className="mt-0">
                <CreateSecret />
              </TabsContent>
              <TabsContent value="view" className="mt-0">
                <ViewSecret />
              </TabsContent>
            </Tabs>
          </motion.div>
        </div>
      </main>
    </TabsContext.Provider>
  );
}
