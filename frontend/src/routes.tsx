import { createBrowserRouter, type RouteObject } from "react-router";
import { AppShell } from "@/components/layout/app-shell";
import { AuthLayout } from "@/components/layout/auth-layout";
import { OnboardingLayout } from "@/components/layout/onboarding-layout";
import { StudentShell } from "@/components/layout/student-shell";
import { AdminShell } from "@/components/layout/admin-shell";
import { ProtectedRoute } from "@/components/layout/protected-route";
import { GuestRoute } from "@/components/layout/guest-route";
import { OnboardingGuard } from "@/components/layout/onboarding-guard";
import { AdminGuard } from "@/components/layout/admin-guard";
import { StudentGuard } from "@/components/layout/student-guard";
import { RouteErrorBoundary } from "@/components/layout/route-error-boundary";
import { NotFoundPage } from "@/components/layout/not-found-page";
import { PlaceholderPage } from "@/components/layout/placeholder-page";

// ─── Phase 5: Auth pages ───────────────────────────────────────────────────────
import { Login } from "@/features/auth/login";
import { Register } from "@/features/auth/register";
import { AccountRecovery } from "@/features/auth/account-recovery";
import { EmailVerification } from "@/features/auth/email-verification";
import { AcceptInvitation } from "@/features/auth/accept-invitation";

// ─── Phase 6: Onboarding wizard ───────────────────────────────────────────────
import { OnboardingWizard } from "@/features/onboarding/onboarding-wizard";

// ─── Phase 7: Settings pages ─────────────────────────────────────────────────
import { FamilySettings } from "@/features/settings/family-settings";
import { NotificationPrefs } from "@/features/settings/notification-prefs";
import { SubscriptionUpgrade } from "@/features/settings/subscription-upgrade";
import { AccountSettings } from "@/features/settings/account-settings";
import { PrivacyControls } from "@/features/settings/privacy-controls";

// ─── Phase 5: Legal pages ─────────────────────────────────────────────────────
import { TermsOfService } from "@/features/legal/terms-of-service";
import { PrivacyPolicy } from "@/features/legal/privacy-policy";
import { CommunityGuidelines } from "@/features/legal/community-guidelines";

// ─── Lazy placeholder factory ────────────────────────────────────────────────
// Each domain page will be replaced with a real lazy(() => import(...)) as it's built.
// For now, every route gets a lightweight placeholder with the correct page title.
function p(title: string) {
  return { element: <PlaceholderPage title={title} /> };
}

const routes: RouteObject[] = [
  // ─── Authenticated routes ──────────────────────────────────────────────────
  {
    element: <ProtectedRoute />,
    errorElement: <RouteErrorBoundary />,
    children: [
      {
        element: <OnboardingGuard />,
        children: [
          {
            element: <AppShell />,
            errorElement: <RouteErrorBoundary />,
            children: [
              // Home / Feed
              { index: true, ...p("Home") },

              // Social
              { path: "friends", ...p("Friends"), errorElement: <RouteErrorBoundary /> },
              { path: "messages", ...p("Messages"), errorElement: <RouteErrorBoundary /> },
              { path: "messages/:conversationId", ...p("Conversation") },
              { path: "groups", ...p("Groups"), errorElement: <RouteErrorBoundary /> },
              { path: "groups/:groupId", ...p("Group") },
              { path: "events", ...p("Events"), errorElement: <RouteErrorBoundary /> },

              // Learning
              { path: "learning", ...p("Learning Dashboard"), errorElement: <RouteErrorBoundary /> },
              { path: "learning/activities", ...p("Activity Log") },
              { path: "learning/journals", ...p("Journals") },
              { path: "learning/journals/new", ...p("New Journal Entry") },
              { path: "learning/reading-lists", ...p("Reading Lists") },
              { path: "learning/progress/:studentId", ...p("Student Progress") },
              { path: "learning/quiz/:sessionId", ...p("Quiz") },
              { path: "learning/video/:videoId", ...p("Video Player") },
              { path: "learning/read/:contentId", ...p("Content Viewer") },
              { path: "learning/sequence/:progressId", ...p("Sequence View") },

              // Marketplace
              { path: "marketplace", ...p("Marketplace"), errorElement: <RouteErrorBoundary /> },
              { path: "marketplace/listings/:id", ...p("Listing Details") },
              { path: "marketplace/cart", ...p("Cart") },
              { path: "marketplace/purchases", ...p("Purchase History") },
              { path: "marketplace/purchases/:id/refund", ...p("Request Refund") },

              // Creator
              { path: "creator", ...p("Creator Dashboard"), errorElement: <RouteErrorBoundary /> },
              { path: "creator/listings/new", ...p("Create Listing") },
              { path: "creator/listings/:id/edit", ...p("Edit Listing") },
              { path: "creator/quiz-builder", ...p("Quiz Builder") },
              { path: "creator/quiz-builder/:id", ...p("Edit Quiz") },
              { path: "creator/sequence-builder", ...p("Sequence Builder") },
              { path: "creator/sequence-builder/:id", ...p("Edit Sequence") },
              { path: "creator/payouts", ...p("Payout Setup") },

              // Settings
              { path: "settings", element: <FamilySettings />, errorElement: <RouteErrorBoundary /> },
              { path: "settings/notifications", element: <NotificationPrefs /> },
              { path: "settings/subscription", element: <SubscriptionUpgrade /> },
              { path: "settings/account", element: <AccountSettings /> },
              { path: "settings/account/sessions", ...p("Session Management") },
              { path: "settings/account/export", ...p("Data Export") },
              { path: "settings/account/delete", ...p("Delete Account") },
              { path: "settings/account/appeals", ...p("Moderation Appeals") },
              { path: "settings/privacy", element: <PrivacyControls /> },

              // Search
              { path: "search", ...p("Search"), errorElement: <RouteErrorBoundary /> },

              // Family profile
              { path: "family/:familyId", ...p("Family Profile") },

              // Calendar / Planning
              { path: "calendar", ...p("Calendar"), errorElement: <RouteErrorBoundary /> },
              { path: "calendar/day/:date", ...p("Day View") },
              { path: "calendar/week/:date", ...p("Week View") },
              { path: "planning/templates", ...p("Schedule Templates") },




              // Notifications (not in route spec but needed for bell icon link)
              { path: "notifications", ...p("Notifications") },

              // 404
              { path: "*", element: <NotFoundPage /> },
            ],
          },
        ],
      },

      // Onboarding (protected but uses its own layout, not OnboardingGuard)
      {
        path: "onboarding",
        element: <OnboardingLayout />,
        errorElement: <RouteErrorBoundary />,
        children: [
          { index: true, element: <OnboardingWizard />, errorElement: <RouteErrorBoundary /> },
        ],
      },
    ],
  },

  // ─── Auth routes (unauthenticated) ─────────────────────────────────────────
  {
    path: "auth",
    element: <AuthLayout />,
    errorElement: <RouteErrorBoundary />,
    children: [
      // Login and register redirect authenticated users away (GuestRoute)
      {
        element: <GuestRoute />,
        children: [
          { path: "login", element: <Login />, errorElement: <RouteErrorBoundary /> },
          { path: "register", element: <Register />, errorElement: <RouteErrorBoundary /> },
        ],
      },
      { path: "recovery", element: <AccountRecovery />, errorElement: <RouteErrorBoundary /> },
      { path: "verification", element: <EmailVerification />, errorElement: <RouteErrorBoundary /> },
      {
        path: "accept-invite/:token",
        element: <AcceptInvitation />,
        errorElement: <RouteErrorBoundary />,
      },
    ],
  },

  // ─── Student routes ────────────────────────────────────────────────────────
  {
    element: <ProtectedRoute />,
    errorElement: <RouteErrorBoundary />,
    children: [
      {
        path: "student",
        element: <StudentGuard />,
        children: [
          {
            element: <StudentShell />,
            errorElement: <RouteErrorBoundary />,
            children: [
              { index: true, ...p("Student Dashboard") },
              { path: "quiz/:sessionId", ...p("Student Quiz") },
              { path: "video/:videoId", ...p("Student Video") },
              { path: "read/:contentId", ...p("Student Reader") },
              { path: "sequence/:progressId", ...p("Student Sequence") },
            ],
          },
        ],
      },
    ],
  },

  // ─── Admin routes ──────────────────────────────────────────────────────────
  {
    element: <ProtectedRoute />,
    errorElement: <RouteErrorBoundary />,
    children: [
      {
        path: "admin",
        element: <AdminGuard />,
        children: [
          {
            element: <AdminShell />,
            errorElement: <RouteErrorBoundary />,
            children: [
              { index: true, ...p("Admin Dashboard") },
              { path: "users", ...p("User Management") },
              { path: "users/:id", ...p("User Details") },
              { path: "moderation", ...p("Moderation Queue") },
              { path: "flags", ...p("Feature Flags") },
              { path: "audit", ...p("Audit Log") },
              { path: "methodologies", ...p("Methodology Config") },
            ],
          },
        ],
      },
    ],
  },

  // ─── Compliance routes ─────────────────────────────────────────────────────
  {
    element: <ProtectedRoute />,
    errorElement: <RouteErrorBoundary />,
    children: [
      {
        path: "compliance",
        element: <AppShell />,
        errorElement: <RouteErrorBoundary />,
        children: [
          { index: true, ...p("Compliance Setup") },
          { path: "attendance", ...p("Attendance Tracker") },
          { path: "assessments", ...p("Assessment Records") },
          { path: "tests", ...p("Standardized Tests") },
          { path: "portfolios", ...p("Portfolios") },
          { path: "portfolios/:id", ...p("Portfolio Builder") },
          { path: "transcripts", ...p("Transcripts") },
          { path: "transcripts/:id", ...p("Transcript Builder") },
        ],
      },
    ],
  },
  // ─── Public legal routes (no auth required — linked from register, reports) ──
  {
    path: "legal",
    errorElement: <RouteErrorBoundary />,
    children: [
      { path: "terms", element: <TermsOfService /> },
      { path: "privacy", element: <PrivacyPolicy /> },
      { path: "guidelines", element: <CommunityGuidelines /> },
    ],
  },
];

export const router = createBrowserRouter(routes);
