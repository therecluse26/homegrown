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
import { SessionManagement } from "@/features/settings/session-management";
import { DataExport } from "@/features/settings/data-export";
import { AccountDeletion } from "@/features/settings/account-deletion";
import { StudentDeletion } from "@/features/settings/student-deletion";
import { NotificationCenter } from "@/features/settings/notification-center";
import { NotificationHistory } from "@/features/settings/notification-history";

// ─── Phase 8: Learning pages ─────────────────────────────────────────────────
import { LearningDashboard } from "@/features/learning/learning-dashboard";
import { ActivityLog } from "@/features/learning/activity-log";
import { JournalList } from "@/features/learning/journal-list";
import { JournalEditor } from "@/features/learning/journal-editor";
import { ReadingLists } from "@/features/learning/reading-lists";
import { ProgressView } from "@/features/learning/progress-view";
import { QuizPlayer } from "@/features/learning/quiz-player";
import { ParentQuizScoring } from "@/features/learning/parent-quiz-scoring";
import { VideoPlayer } from "@/features/learning/video-player";
import { ContentViewer } from "@/features/learning/content-viewer";
import { SequenceView } from "@/features/learning/sequence-view";
import { TestsAndGrades } from "@/features/learning/tests-and-grades";
import { StudentSessionActivityLog } from "@/features/learning/student-session-activity-log";
import { StudentSessionLauncher } from "@/features/learning/student-session-launcher";

// ─── Phase 8: Student pages ─────────────────────────────────────────────────
import { StudentDashboard as StudentDashboardPage } from "@/features/student/student-dashboard";
import { StudentQuiz } from "@/features/student/student-quiz";
import { StudentVideo } from "@/features/student/student-video";
import { StudentReader } from "@/features/student/student-reader";
import { StudentSequence } from "@/features/student/student-sequence";

// ─── Phase 9: Social pages ──────────────────────────────────────────────────
import { Feed } from "@/features/social/feed";
import { FriendsList } from "@/features/social/friends-list";
import { DirectMessages } from "@/features/social/direct-messages";
import { Conversation as ConversationPage } from "@/features/social/conversation";
import { GroupsList } from "@/features/social/groups-list";
import { GroupDetail } from "@/features/social/group-detail";
import { EventsList } from "@/features/social/events-list";
import { FamilyProfile } from "@/features/social/family-profile";
import { PostDetail } from "@/features/social/post-detail";

// ─── Phase 9: Marketplace pages ────────────────────────────────────────────
import { MarketplaceBrowse } from "@/features/marketplace/marketplace-browse";
import { ListingDetail } from "@/features/marketplace/listing-detail";
import { Cart } from "@/features/marketplace/cart";
import { PurchaseHistory } from "@/features/marketplace/purchase-history";
import { RefundRequest } from "@/features/marketplace/refund-request";
import { CreatorDashboard } from "@/features/marketplace/creator/creator-dashboard";
import { CreateListing } from "@/features/marketplace/creator/create-listing";
import { EditListing } from "@/features/marketplace/creator/edit-listing";
import { QuizBuilder } from "@/features/marketplace/creator/quiz-builder";
import { SequenceBuilder } from "@/features/marketplace/creator/sequence-builder";

// ─── Phase 9: Search pages ──────────────────────────────────────────────────
import { SearchResults } from "@/features/search/search-results";

// ─── Phase 9: Admin pages ──────────────────────────────────────────────────
import { AdminDashboard } from "@/features/admin/admin-dashboard";
import { UserManagement } from "@/features/admin/user-management";
import { UserDetail } from "@/features/admin/user-detail";
import { ModerationQueue } from "@/features/admin/moderation-queue";
import { AuditLog } from "@/features/admin/audit-log";

// ─── Phase 9: Settings additions ────────────────────────────────────────────
import { ModerationAppeals } from "@/features/settings/moderation-appeals";
import { BlockManagement } from "@/features/settings/block-management";

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
              { index: true, element: <Feed />, errorElement: <RouteErrorBoundary /> },

              // Social
              { path: "friends", element: <FriendsList />, errorElement: <RouteErrorBoundary /> },
              { path: "messages", element: <DirectMessages />, errorElement: <RouteErrorBoundary /> },
              { path: "messages/:conversationId", element: <ConversationPage /> },
              { path: "groups", element: <GroupsList />, errorElement: <RouteErrorBoundary /> },
              { path: "groups/:groupId", element: <GroupDetail /> },
              { path: "events", element: <EventsList />, errorElement: <RouteErrorBoundary /> },
              { path: "post/:postId", element: <PostDetail /> },

              // Learning
              { path: "learning", element: <LearningDashboard />, errorElement: <RouteErrorBoundary /> },
              { path: "learning/activities", element: <ActivityLog /> },
              { path: "learning/journals", element: <JournalList /> },
              { path: "learning/journals/new", element: <JournalEditor /> },
              { path: "learning/reading-lists", element: <ReadingLists /> },
              { path: "learning/progress/:studentId", element: <ProgressView /> },
              { path: "learning/grades", element: <TestsAndGrades /> },
              { path: "learning/quiz/:sessionId", element: <QuizPlayer /> },
              { path: "learning/quiz/:sessionId/score", element: <ParentQuizScoring /> },
              { path: "learning/video/:videoId", element: <VideoPlayer /> },
              { path: "learning/read/:contentId", element: <ContentViewer /> },
              { path: "learning/sequence/:progressId", element: <SequenceView /> },
              { path: "learning/session-log/:sessionId", element: <StudentSessionActivityLog /> },
              { path: "learning/session", element: <StudentSessionLauncher /> },

              // Marketplace
              { path: "marketplace", element: <MarketplaceBrowse />, errorElement: <RouteErrorBoundary /> },
              { path: "marketplace/listings/:id", element: <ListingDetail /> },
              { path: "marketplace/cart", element: <Cart /> },
              { path: "marketplace/purchases", element: <PurchaseHistory /> },
              { path: "marketplace/purchases/:id/refund", element: <RefundRequest /> },

              // Creator
              { path: "creator", element: <CreatorDashboard />, errorElement: <RouteErrorBoundary /> },
              { path: "creator/listings/new", element: <CreateListing /> },
              { path: "creator/listings/:id/edit", element: <EditListing /> },
              { path: "creator/quiz-builder", element: <QuizBuilder /> },
              { path: "creator/quiz-builder/:id", element: <QuizBuilder /> },
              { path: "creator/sequence-builder", element: <SequenceBuilder /> },
              { path: "creator/sequence-builder/:id", element: <SequenceBuilder /> },
              { path: "creator/payouts", ...p("Payout Setup") },

              // Settings
              { path: "settings", element: <FamilySettings />, errorElement: <RouteErrorBoundary /> },
              { path: "settings/notifications", element: <NotificationPrefs /> },
              { path: "settings/notifications/history", element: <NotificationHistory /> },
              { path: "settings/subscription", element: <SubscriptionUpgrade /> },
              { path: "settings/account", element: <AccountSettings /> },
              { path: "settings/account/sessions", element: <SessionManagement /> },
              { path: "settings/account/export", element: <DataExport /> },
              { path: "settings/account/delete", element: <AccountDeletion /> },
              { path: "settings/account/delete/student/:studentId", element: <StudentDeletion /> },
              { path: "settings/account/appeals", element: <ModerationAppeals /> },
              { path: "settings/blocks", element: <BlockManagement /> },
              { path: "settings/privacy", element: <PrivacyControls /> },

              // Search
              { path: "search", element: <SearchResults />, errorElement: <RouteErrorBoundary /> },

              // Family profile
              { path: "family/:familyId", element: <FamilyProfile /> },

              // Calendar / Planning
              { path: "calendar", ...p("Calendar"), errorElement: <RouteErrorBoundary /> },
              { path: "calendar/day/:date", ...p("Day View") },
              { path: "calendar/week/:date", ...p("Week View") },
              { path: "planning/templates", ...p("Schedule Templates") },




              // Notifications
              { path: "notifications", element: <NotificationCenter /> },

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
              { index: true, element: <StudentDashboardPage /> },
              { path: "quiz/:sessionId", element: <StudentQuiz /> },
              { path: "video/:videoId", element: <StudentVideo /> },
              { path: "read/:contentId", element: <StudentReader /> },
              { path: "sequence/:progressId", element: <StudentSequence /> },
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
              { index: true, element: <AdminDashboard /> },
              { path: "users", element: <UserManagement /> },
              { path: "users/:id", element: <UserDetail /> },
              { path: "moderation", element: <ModerationQueue /> },
              { path: "flags", ...p("Feature Flags") },
              { path: "audit", element: <AuditLog /> },
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
