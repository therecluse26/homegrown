import type { components } from "@/api/generated/schema";

// ─── Auth / IAM ──────────────────────────────────────────────────────────────

export type CurrentUser = components["schemas"]["iam.CurrentUserResponse"];
export type FamilyProfile = components["schemas"]["iam.FamilyProfileResponse"];
export type Student = components["schemas"]["iam.StudentResponse"];

// ─── Onboarding ──────────────────────────────────────────────────────────────

export type WizardProgress = components["schemas"]["onboard.WizardProgressResponse"];
export type WizardStatus = components["schemas"]["onboard.WizardStatus"];

// ─── Family / Co-Parents ────────────────────────────────────────────────────

export type ParentSummary = components["schemas"]["iam.ParentSummary"];
export type CoParentInvite = components["schemas"]["iam.CoParentInviteResponse"];
export type ActiveTool = components["schemas"]["method.ActiveToolResponse"];
export type MethodologySelection =
  components["schemas"]["method.MethodologySelectionResponse"];

// ─── Shared ──────────────────────────────────────────────────────────────────

export type UserContext = "parent" | "student";
