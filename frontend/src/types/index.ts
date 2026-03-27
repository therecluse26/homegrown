import type { components } from "@/api/generated/schema";

// ─── Auth / IAM ──────────────────────────────────────────────────────────────

export type CurrentUser = components["schemas"]["iam.CurrentUserResponse"];
export type FamilyProfile = components["schemas"]["iam.FamilyProfileResponse"];
export type Student = components["schemas"]["iam.StudentResponse"];

// ─── Onboarding ──────────────────────────────────────────────────────────────

export type WizardProgress = components["schemas"]["onboard.WizardProgressResponse"];
export type WizardStatus = components["schemas"]["onboard.WizardStatus"];

// ─── Shared ──────────────────────────────────────────────────────────────────

export type UserContext = "parent" | "student";
