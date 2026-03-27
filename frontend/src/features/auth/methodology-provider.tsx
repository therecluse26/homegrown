import { createContext, useContext, type ReactNode } from "react";
import { useMethodology } from "@/hooks/use-methodology";
import type { components } from "@/api/generated/schema";

type ActiveTool = components["schemas"]["method.ActiveToolResponse"];

type MethodologyContextValue = {
  isLoading: boolean;
  primarySlug: string | null;
  primaryName: string | null;
  secondarySlugs: (string | undefined)[];
  terminology: Record<string, never>;
  tools: ActiveTool[];
};

const MethodologyContext = createContext<MethodologyContextValue | null>(null);

export function MethodologyProvider({ children }: { children: ReactNode }) {
  const methodology = useMethodology();
  return (
    <MethodologyContext.Provider value={methodology}>
      {children}
    </MethodologyContext.Provider>
  );
}

export function useMethodologyContext() {
  const ctx = useContext(MethodologyContext);
  if (!ctx) {
    throw new Error("useMethodologyContext must be used within <MethodologyProvider>");
  }
  return ctx;
}
