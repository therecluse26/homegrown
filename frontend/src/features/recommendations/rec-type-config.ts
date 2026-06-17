import { BookOpen, Lightbulb, Package, Sparkles } from "lucide-react";

export const REC_TYPES = [
  "marketplace_content",
  "activity_idea",
  "reading_suggestion",
  "community_group",
] as const;

export type RecType = (typeof REC_TYPES)[number];

export const TYPE_CONFIG: Record<
  RecType,
  {
    icon: typeof BookOpen;
    badgeVariant: "primary" | "secondary" | "success";
    labelId: string;
  }
> = {
  marketplace_content: {
    icon: BookOpen,
    badgeVariant: "primary",
    labelId: "recommendations.type.content",
  },
  activity_idea: {
    icon: Lightbulb,
    badgeVariant: "secondary",
    labelId: "recommendations.type.activity",
  },
  reading_suggestion: {
    icon: Package,
    badgeVariant: "success",
    labelId: "recommendations.type.resource",
  },
  community_group: {
    icon: Sparkles,
    badgeVariant: "primary",
    labelId: "recommendations.type.community",
  },
};

export const DEFAULT_REC_CONFIG = {
  icon: BookOpen,
  badgeVariant: "primary" as const,
  labelId: "recommendations.type.content",
};
