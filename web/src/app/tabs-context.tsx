import { createContext } from "react";

interface TabsContextType {
  activeTab: string;
  setActiveTab: (tab: string) => void;
}

export const TabsContext = createContext<TabsContextType>({
  activeTab: "create",
  setActiveTab: () => {},
});
