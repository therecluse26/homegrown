import { lazy } from "react";
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

// ─── Lazy-loaded feature pages ──────────────────────────────────────────────
// Each feature module is code-split into its own chunk via React.lazy.
// Suspense boundaries in each shell component (AppShell, AdminShell, etc.)
// show a spinner while the chunk loads.

// Auth
const Login = lazy(() => import("@/features/auth/login").then(m => ({ default: m.Login })));
const Register = lazy(() => import("@/features/auth/register").then(m => ({ default: m.Register })));
const AccountRecovery = lazy(() => import("@/features/auth/account-recovery").then(m => ({ default: m.AccountRecovery })));
const EmailVerification = lazy(() => import("@/features/auth/email-verification").then(m => ({ default: m.EmailVerification })));
const AcceptInvitation = lazy(() => import("@/features/auth/accept-invitation").then(m => ({ default: m.AcceptInvitation })));

// Onboarding
const OnboardingWizard = lazy(() => import("@/features/onboarding/onboarding-wizard").then(m => ({ default: m.OnboardingWizard })));

// Settings
const FamilySettings = lazy(() => import("@/features/settings/family-settings").then(m => ({ default: m.FamilySettings })));
const NotificationPrefs = lazy(() => import("@/features/settings/notification-prefs").then(m => ({ default: m.NotificationPrefs })));
const SubscriptionUpgrade = lazy(() => import("@/features/settings/subscription-upgrade").then(m => ({ default: m.SubscriptionUpgrade })));
const AccountSettings = lazy(() => import("@/features/settings/account-settings").then(m => ({ default: m.AccountSettings })));
const PrivacyControls = lazy(() => import("@/features/settings/privacy-controls").then(m => ({ default: m.PrivacyControls })));
const SessionManagement = lazy(() => import("@/features/settings/session-management").then(m => ({ default: m.SessionManagement })));
const DataExport = lazy(() => import("@/features/settings/data-export").then(m => ({ default: m.DataExport })));
const AccountDeletion = lazy(() => import("@/features/settings/account-deletion").then(m => ({ default: m.AccountDeletion })));
const StudentDeletion = lazy(() => import("@/features/settings/student-deletion").then(m => ({ default: m.StudentDeletion })));
const NotificationCenter = lazy(() => import("@/features/settings/notification-center").then(m => ({ default: m.NotificationCenter })));
const NotificationHistory = lazy(() => import("@/features/settings/notification-history").then(m => ({ default: m.NotificationHistory })));
const ModerationAppeals = lazy(() => import("@/features/settings/moderation-appeals").then(m => ({ default: m.ModerationAppeals })));
const BlockManagement = lazy(() => import("@/features/settings/block-management").then(m => ({ default: m.BlockManagement })));

// Learning
const LearningDashboard = lazy(() => import("@/features/learning/learning-dashboard").then(m => ({ default: m.LearningDashboard })));
const ActivityLog = lazy(() => import("@/features/learning/activity-log").then(m => ({ default: m.ActivityLog })));
const JournalList = lazy(() => import("@/features/learning/journal-list").then(m => ({ default: m.JournalList })));
const JournalEditor = lazy(() => import("@/features/learning/journal-editor").then(m => ({ default: m.JournalEditor })));
const ReadingLists = lazy(() => import("@/features/learning/reading-lists").then(m => ({ default: m.ReadingLists })));
const ProgressView = lazy(() => import("@/features/learning/progress-view").then(m => ({ default: m.ProgressView })));
const QuizPlayer = lazy(() => import("@/features/learning/quiz-player").then(m => ({ default: m.QuizPlayer })));
const ParentQuizScoring = lazy(() => import("@/features/learning/parent-quiz-scoring").then(m => ({ default: m.ParentQuizScoring })));
const VideoPlayer = lazy(() => import("@/features/learning/video-player").then(m => ({ default: m.VideoPlayer })));
const ContentViewer = lazy(() => import("@/features/learning/content-viewer").then(m => ({ default: m.ContentViewer })));
const SequenceView = lazy(() => import("@/features/learning/sequence-view").then(m => ({ default: m.SequenceView })));
const TestsAndGrades = lazy(() => import("@/features/learning/tests-and-grades").then(m => ({ default: m.TestsAndGrades })));
const StudentSessionActivityLog = lazy(() => import("@/features/learning/student-session-activity-log").then(m => ({ default: m.StudentSessionActivityLog })));
const StudentSessionLauncher = lazy(() => import("@/features/learning/student-session-launcher").then(m => ({ default: m.StudentSessionLauncher })));

// Student
const StudentDashboardPage = lazy(() => import("@/features/student/student-dashboard").then(m => ({ default: m.StudentDashboard })));
const StudentQuiz = lazy(() => import("@/features/student/student-quiz").then(m => ({ default: m.StudentQuiz })));
const StudentVideo = lazy(() => import("@/features/student/student-video").then(m => ({ default: m.StudentVideo })));
const StudentReader = lazy(() => import("@/features/student/student-reader").then(m => ({ default: m.StudentReader })));
const StudentSequence = lazy(() => import("@/features/student/student-sequence").then(m => ({ default: m.StudentSequence })));

// Social
const Feed = lazy(() => import("@/features/social/feed").then(m => ({ default: m.Feed })));
const FriendsList = lazy(() => import("@/features/social/friends-list").then(m => ({ default: m.FriendsList })));
const FriendDiscovery = lazy(() => import("@/features/social/friend-discovery").then(m => ({ default: m.FriendDiscovery })));
const DirectMessages = lazy(() => import("@/features/social/direct-messages").then(m => ({ default: m.DirectMessages })));
const ConversationPage = lazy(() => import("@/features/social/conversation").then(m => ({ default: m.Conversation })));
const GroupsList = lazy(() => import("@/features/social/groups-list").then(m => ({ default: m.GroupsList })));
const GroupDetail = lazy(() => import("@/features/social/group-detail").then(m => ({ default: m.GroupDetail })));
const EventsList = lazy(() => import("@/features/social/events-list").then(m => ({ default: m.EventsList })));
const EventCreation = lazy(() => import("@/features/social/event-creation").then(m => ({ default: m.EventCreation })));
const EventDetail = lazy(() => import("@/features/social/event-detail").then(m => ({ default: m.EventDetail })));
const FamilyProfile = lazy(() => import("@/features/social/family-profile").then(m => ({ default: m.FamilyProfile })));
const PostDetail = lazy(() => import("@/features/social/post-detail").then(m => ({ default: m.PostDetail })));

// Marketplace
const MarketplaceBrowse = lazy(() => import("@/features/marketplace/marketplace-browse").then(m => ({ default: m.MarketplaceBrowse })));
const ListingDetail = lazy(() => import("@/features/marketplace/listing-detail").then(m => ({ default: m.ListingDetail })));
const Cart = lazy(() => import("@/features/marketplace/cart").then(m => ({ default: m.Cart })));
const PurchaseHistory = lazy(() => import("@/features/marketplace/purchase-history").then(m => ({ default: m.PurchaseHistory })));
const RefundRequest = lazy(() => import("@/features/marketplace/refund-request").then(m => ({ default: m.RefundRequest })));
const CreatorDashboard = lazy(() => import("@/features/marketplace/creator/creator-dashboard").then(m => ({ default: m.CreatorDashboard })));
const CreateListing = lazy(() => import("@/features/marketplace/creator/create-listing").then(m => ({ default: m.CreateListing })));
const EditListing = lazy(() => import("@/features/marketplace/creator/edit-listing").then(m => ({ default: m.EditListing })));
const QuizBuilder = lazy(() => import("@/features/marketplace/creator/quiz-builder").then(m => ({ default: m.QuizBuilder })));
const SequenceBuilder = lazy(() => import("@/features/marketplace/creator/sequence-builder").then(m => ({ default: m.SequenceBuilder })));

// Search
const SearchResults = lazy(() => import("@/features/search/search-results").then(m => ({ default: m.SearchResults })));

// Admin
const AdminDashboard = lazy(() => import("@/features/admin/admin-dashboard").then(m => ({ default: m.AdminDashboard })));
const UserManagement = lazy(() => import("@/features/admin/user-management").then(m => ({ default: m.UserManagement })));
const UserDetail = lazy(() => import("@/features/admin/user-detail").then(m => ({ default: m.UserDetail })));
const ModerationQueue = lazy(() => import("@/features/admin/moderation-queue").then(m => ({ default: m.ModerationQueue })));
const AuditLog = lazy(() => import("@/features/admin/audit-log").then(m => ({ default: m.AuditLog })));

// Planning
const CalendarView = lazy(() => import("@/features/planning/calendar-view").then(m => ({ default: m.CalendarView })));
const ScheduleEditor = lazy(() => import("@/features/planning/schedule-editor").then(m => ({ default: m.ScheduleEditor })));

// Legal
const TermsOfService = lazy(() => import("@/features/legal/terms-of-service").then(m => ({ default: m.TermsOfService })));
const PrivacyPolicy = lazy(() => import("@/features/legal/privacy-policy").then(m => ({ default: m.PrivacyPolicy })));
const CommunityGuidelines = lazy(() => import("@/features/legal/community-guidelines").then(m => ({ default: m.CommunityGuidelines })));

// ─── Lazy placeholder factory ────────────────────────────────────────────────
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
              { path: "friends/discover", element: <FriendDiscovery /> },
              { path: "messages", element: <DirectMessages />, errorElement: <RouteErrorBoundary /> },
              { path: "messages/:conversationId", element: <ConversationPage /> },
              { path: "groups", element: <GroupsList />, errorElement: <RouteErrorBoundary /> },
              { path: "groups/:groupId", element: <GroupDetail /> },
              { path: "events", element: <EventsList />, errorElement: <RouteErrorBoundary /> },
              { path: "events/new", element: <EventCreation /> },
              { path: "events/:eventId", element: <EventDetail /> },
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
              { path: "calendar", element: <CalendarView />, errorElement: <RouteErrorBoundary /> },
              { path: "calendar/day/:date", element: <CalendarView /> },
              { path: "calendar/week/:date", element: <CalendarView /> },
              { path: "schedule/new", element: <ScheduleEditor /> },
              { path: "schedule/:itemId/edit", element: <ScheduleEditor /> },
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
