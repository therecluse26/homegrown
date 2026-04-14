import { lazy } from "react";
import { createBrowserRouter, Navigate, useParams, type RouteObject } from "react-router";
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

/** Redirect that interpolates route params into the target path. */
function ParamRedirect({ to }: { to: string }) {
  const params = useParams();
  let target = to;
  for (const [key, value] of Object.entries(params)) {
    if (value && key !== "*") target = target.replace(`:${key}`, value);
  }
  return <Navigate to={target} replace />;
}

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
const CoppaMicroCharge = lazy(() => import("@/features/auth/coppa-micro-charge").then(m => ({ default: m.CoppaMicroCharge })));

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
const MfaSetup = lazy(() => import("@/features/settings/mfa-setup").then(m => ({ default: m.MfaSetup })));
const SubscriptionManager = lazy(() => import("@/features/settings/subscription-manager").then(m => ({ default: m.SubscriptionManager })));

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
const Projects = lazy(() => import("@/features/learning/projects").then(m => ({ default: m.Projects })));
const ToolAssignment = lazy(() => import("@/features/learning/tool-assignment").then(m => ({ default: m.ToolAssignment })));
const NatureJournal = lazy(() => import("@/features/learning/nature-journal").then(m => ({ default: m.NatureJournal })));
const TriviumTracker = lazy(() => import("@/features/learning/trivium-tracker").then(m => ({ default: m.TriviumTracker })));
const RhythmPlanner = lazy(() => import("@/features/learning/rhythm-planner").then(m => ({ default: m.RhythmPlanner })));
const ObservationLogs = lazy(() => import("@/features/learning/observation-logs").then(m => ({ default: m.ObservationLogs })));
const HabitTracking = lazy(() => import("@/features/learning/habit-tracking").then(m => ({ default: m.HabitTracking })));
const InterestLedLog = lazy(() => import("@/features/learning/interest-led-log").then(m => ({ default: m.InterestLedLog })));
const HandworkProjects = lazy(() => import("@/features/learning/handwork-projects").then(m => ({ default: m.HandworkProjects })));
const PracticalLife = lazy(() => import("@/features/learning/practical-life").then(m => ({ default: m.PracticalLife })));

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
const GroupCreation = lazy(() => import("@/features/social/group-creation").then(m => ({ default: m.GroupCreation })));
const GroupManagement = lazy(() => import("@/features/social/group-management").then(m => ({ default: m.GroupManagement })));

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
const PayoutSetup = lazy(() => import("@/features/marketplace/payout-setup").then(m => ({ default: m.PayoutSetup })));
const CreatorVerification = lazy(() => import("@/features/marketplace/creator-verification").then(m => ({ default: m.CreatorVerification })));
const CreatorReviews = lazy(() => import("@/features/marketplace/creator-reviews").then(m => ({ default: m.CreatorReviews })));
const ListingVersionHistory = lazy(() => import("@/features/marketplace/listing-version-history").then(m => ({ default: m.ListingVersionHistory })));

// Stub / redirect pages
const ComingSoonStub = lazy(() => import("@/components/common/coming-soon-stub").then(m => ({ default: m.ComingSoonStub })));
const ProfileRedirect = lazy(() => import("@/features/social/profile-redirect").then(m => ({ default: m.ProfileRedirect })));
const LogoutRoute = lazy(() => import("@/features/auth/logout-route").then(m => ({ default: m.LogoutRoute })));

// Billing
const PricingPage = lazy(() => import("@/features/billing/pricing-page").then(m => ({ default: m.PricingPage })));
const PaymentMethods = lazy(() => import("@/features/billing/payment-methods").then(m => ({ default: m.PaymentMethods })));
const TransactionHistory = lazy(() => import("@/features/billing/transaction-history").then(m => ({ default: m.TransactionHistory })));
const SubscriptionManagement = lazy(() => import("@/features/billing/subscription-management").then(m => ({ default: m.SubscriptionManagement })));
const InvoiceHistory = lazy(() => import("@/features/billing/invoice-history").then(m => ({ default: m.InvoiceHistory })));

// Recommendations
const RecommendationsPage = lazy(() => import("@/features/recommendations/recommendations-page").then(m => ({ default: m.RecommendationsPage })));

// Search
const SearchResults = lazy(() => import("@/features/search/search-results").then(m => ({ default: m.SearchResults })));

// Admin
const AdminDashboard = lazy(() => import("@/features/admin/admin-dashboard").then(m => ({ default: m.AdminDashboard })));
const UserManagement = lazy(() => import("@/features/admin/user-management").then(m => ({ default: m.UserManagement })));
const UserDetail = lazy(() => import("@/features/admin/user-detail").then(m => ({ default: m.UserDetail })));
const ModerationQueue = lazy(() => import("@/features/admin/moderation-queue").then(m => ({ default: m.ModerationQueue })));
const AuditLog = lazy(() => import("@/features/admin/audit-log").then(m => ({ default: m.AuditLog })));
const FeatureFlags = lazy(() => import("@/features/admin/feature-flags").then(m => ({ default: m.FeatureFlags })));
const MethodologyConfigPage = lazy(() => import("@/features/admin/methodology-config").then(m => ({ default: m.MethodologyConfig })));

// Planning
const CalendarView = lazy(() => import("@/features/planning/calendar-view").then(m => ({ default: m.CalendarView })));
const ScheduleEditor = lazy(() => import("@/features/planning/schedule-editor").then(m => ({ default: m.ScheduleEditor })));
const ScheduleTemplates = lazy(() => import("@/features/planning/schedule-templates").then(m => ({ default: m.ScheduleTemplates })));
const SchedulePrint = lazy(() => import("@/features/planning/schedule-print").then(m => ({ default: m.SchedulePrint })));
const CoopCoordination = lazy(() => import("@/features/planning/coop-coordination").then(m => ({ default: m.CoopCoordination })));

// Compliance
const ComplianceSetup = lazy(() => import("@/features/compliance/compliance-setup").then(m => ({ default: m.ComplianceSetup })));
const AttendanceTracker = lazy(() => import("@/features/compliance/attendance-tracker").then(m => ({ default: m.AttendanceTracker })));
const AssessmentRecords = lazy(() => import("@/features/compliance/assessment-records").then(m => ({ default: m.AssessmentRecords })));
const StandardizedTests = lazy(() => import("@/features/compliance/standardized-tests").then(m => ({ default: m.StandardizedTests })));
const PortfolioList = lazy(() => import("@/features/compliance/portfolio-list").then(m => ({ default: m.PortfolioList })));
const PortfolioBuilder = lazy(() => import("@/features/compliance/portfolio-builder").then(m => ({ default: m.PortfolioBuilder })));
const TranscriptList = lazy(() => import("@/features/compliance/transcript-list").then(m => ({ default: m.TranscriptList })));
const TranscriptBuilder = lazy(() => import("@/features/compliance/transcript-builder").then(m => ({ default: m.TranscriptBuilder })));

// Legal
const TermsOfService = lazy(() => import("@/features/legal/terms-of-service").then(m => ({ default: m.TermsOfService })));
const PrivacyPolicy = lazy(() => import("@/features/legal/privacy-policy").then(m => ({ default: m.PrivacyPolicy })));
const CommunityGuidelines = lazy(() => import("@/features/legal/community-guidelines").then(m => ({ default: m.CommunityGuidelines })));

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
              { path: "groups/new", element: <GroupCreation /> },
              { path: "groups/:groupId", element: <GroupDetail /> },
              { path: "groups/:groupId/manage", element: <GroupManagement /> },
              { path: "events", element: <EventsList />, errorElement: <RouteErrorBoundary /> },
              { path: "events/new", element: <EventCreation /> },
              { path: "events/create", element: <Navigate to="/events/new" replace /> },
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
              { path: "learning/projects", element: <Projects /> },
              { path: "learning/tools", element: <ToolAssignment /> },
              // Methodology-specific tools [P3]
              { path: "learning/nature-journal", element: <NatureJournal /> },
              { path: "learning/trivium-tracker", element: <TriviumTracker /> },
              { path: "learning/rhythm-planner", element: <RhythmPlanner /> },
              { path: "learning/observation-logs", element: <ObservationLogs /> },
              { path: "learning/habit-tracking", element: <HabitTracking /> },
              { path: "learning/interest-led-log", element: <InterestLedLog /> },
              { path: "learning/handwork-projects", element: <HandworkProjects /> },
              { path: "learning/practical-life", element: <PracticalLife /> },

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
              { path: "creator/payouts", element: <PayoutSetup /> },
              { path: "creator/verification", element: <CreatorVerification /> },
              { path: "creator/reviews", element: <CreatorReviews /> },

              // Marketplace listing versions
              { path: "marketplace/listings/:listingId/versions", element: <ListingVersionHistory /> },

              // Billing
              { path: "billing", element: <PricingPage />, errorElement: <RouteErrorBoundary /> },
              { path: "billing/payment-methods", element: <PaymentMethods /> },
              { path: "billing/transactions", element: <TransactionHistory /> },
              { path: "billing/subscription", element: <SubscriptionManagement /> },
              { path: "billing/invoices", element: <InvoiceHistory /> },

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
              { path: "settings/account/mfa", element: <MfaSetup /> },
              { path: "settings/subscription/manage", element: <SubscriptionManager /> },

              // Recommendations
              { path: "recommendations", element: <RecommendationsPage />, errorElement: <RouteErrorBoundary /> },

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
              { path: "planning/templates", element: <ScheduleTemplates /> },
              { path: "planning/print", element: <SchedulePrint /> },
              { path: "planning/coop", element: <CoopCoordination /> },


              // Notifications
              { path: "notifications", element: <NotificationCenter /> },

              // ─── Stub pages (coming soon) ───────────────────────────────────
              { path: "creator/earnings", element: <ComingSoonStub title="Creator Earnings" description="Track your marketplace earnings and payout history." backTo="/creator" backLabel="Back to Creator Dashboard" /> },
              { path: "creator/analytics", element: <ComingSoonStub title="Creator Analytics" description="View performance metrics for your marketplace listings." backTo="/creator" backLabel="Back to Creator Dashboard" /> },
              { path: "marketplace/library", element: <ComingSoonStub title="My Library" description="Access your purchased resources and downloads." backTo="/marketplace" backLabel="Back to Marketplace" /> },
              // Learning detail stubs
              { path: "learning/journals/:id", element: <ComingSoonStub title="Journal Entry" description="View and edit journal entries." backTo="/learning/journals" backLabel="Back to Journals" /> },
              { path: "learning/nature-journal/:id", element: <ComingSoonStub title="Nature Journal Entry" description="View nature journal observations." backTo="/learning/nature-journal" backLabel="Back to Nature Journal" /> },
              { path: "learning/trivium-tracker/:id", element: <ComingSoonStub title="Trivium Entry" description="View trivium stage details." backTo="/learning/trivium-tracker" backLabel="Back to Trivium Tracker" /> },
              { path: "learning/grades/new", element: <ComingSoonStub title="New Grade Entry" description="Record a new grade or test score." backTo="/learning/grades" backLabel="Back to Grades" /> },
              { path: "learning/grades/:id", element: <ComingSoonStub title="Grade Details" description="View grade and score details." backTo="/learning/grades" backLabel="Back to Grades" /> },
              { path: "learning/reading-lists/:id", element: <ComingSoonStub title="Reading List" description="View and manage this reading list." backTo="/learning/reading-lists" backLabel="Back to Reading Lists" /> },
              { path: "learning/reading-lists/:id/books", element: <ComingSoonStub title="Books" description="Browse books in this reading list." backTo="/learning/reading-lists" backLabel="Back to Reading Lists" /> },
              { path: "learning/activities/new", element: <ComingSoonStub title="Log Activity" description="Record a new learning activity." backTo="/learning/activities" backLabel="Back to Activities" /> },
              { path: "learning/activities/:id", element: <ComingSoonStub title="Activity Details" description="View activity details and notes." backTo="/learning/activities" backLabel="Back to Activities" /> },
              // Marketplace stubs
              { path: "marketplace/categories", element: <ComingSoonStub title="Categories" description="Browse marketplace content by category." backTo="/marketplace" backLabel="Back to Marketplace" /> },
              // Billing stubs
              { path: "billing/micro-charges", element: <ComingSoonStub title="COPPA Micro-Charges" description="View COPPA verification micro-charge history." backTo="/billing" backLabel="Back to Billing" /> },

              // ─── Redirects ──────────────────────────────────────────────────
              { path: "settings/billing", element: <Navigate to="/billing" replace /> },
              { path: "notifications/history", element: <Navigate to="/settings/notifications/history" replace /> },
              { path: "settings/moderation", element: <Navigate to="/settings/account/appeals" replace /> },
              { path: "profile", element: <ProfileRedirect /> },
              // Learning redirects
              { path: "learning/habit-tracker", element: <Navigate to="/learning/habit-tracking" replace /> },
              { path: "learning/interest-led", element: <Navigate to="/learning/interest-led-log" replace /> },
              // Marketplace redirects
              { path: "marketplace/refund/:id", element: <ParamRedirect to="/marketplace/purchases/:id/refund" /> },
              // Creator redirects
              { path: "creator/listings", element: <Navigate to="/creator" replace /> },
              { path: "creator/quizzes", element: <Navigate to="/creator/quiz-builder" replace /> },
              { path: "creator/quizzes/new", element: <Navigate to="/creator/quiz-builder" replace /> },
              { path: "creator/quizzes/:id", element: <ParamRedirect to="/creator/quiz-builder/:id" /> },
              { path: "creator/sequences", element: <Navigate to="/creator/sequence-builder" replace /> },
              { path: "creator/sequences/new", element: <Navigate to="/creator/sequence-builder" replace /> },
              { path: "creator/sequences/:id", element: <ParamRedirect to="/creator/sequence-builder/:id" /> },
              // Billing redirects
              { path: "billing/history", element: <Navigate to="/billing/transactions" replace /> },
              // Settings redirects
              { path: "settings/family", element: <Navigate to="/settings" replace /> },
              { path: "settings/family/students", element: <Navigate to="/settings" replace /> },
              { path: "settings/family/parents", element: <Navigate to="/settings" replace /> },
              { path: "settings/account/privacy", element: <Navigate to="/settings/privacy" replace /> },
              { path: "settings/methodology", element: <Navigate to="/settings" replace /> },
              { path: "settings/blocked", element: <Navigate to="/settings/blocks" replace /> },
              // Calendar/Planning redirects
              { path: "calendar/week", element: <Navigate to="/calendar" replace /> },
              { path: "calendar/month", element: <Navigate to="/calendar" replace /> },
              { path: "calendar/new", element: <Navigate to="/schedule/new" replace /> },
              { path: "calendar/schedules", element: <Navigate to="/calendar" replace /> },
              { path: "calendar/templates", element: <Navigate to="/planning/templates" replace /> },
              { path: "calendar/templates/new", element: <Navigate to="/planning/templates" replace /> },
              { path: "calendar/routines", element: <Navigate to="/calendar" replace /> },

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
      { path: "coppa/verify", element: <CoppaMicroCharge />, errorElement: <RouteErrorBoundary /> },
      {
        path: "accept-invite/:token",
        element: <AcceptInvitation />,
        errorElement: <RouteErrorBoundary />,
      },
      // Logout route — not wrapped in GuestRoute since authenticated users hit it
      { path: "logout", element: <LogoutRoute /> },
    ],
  },

  // ─── Student routes ────────────────────────────────────────────────────────
  {
    element: <ProtectedRoute />,
    errorElement: <RouteErrorBoundary />,
    children: [
      // Student login is outside StudentGuard — it's the entry point
      { path: "student/login", element: <ComingSoonStub title="Student Login" description="Students sign in with their family code to access their dashboard." backTo="/" backLabel="Back to Home" /> },
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
              // Student stubs
              { path: "journal", element: <ComingSoonStub title="My Journal" description="Write and review your journal entries." backTo="/student" backLabel="Back to Dashboard" /> },
              { path: "activities", element: <ComingSoonStub title="My Activities" description="View your learning activities and progress." backTo="/student" backLabel="Back to Dashboard" /> },
              { path: "reading-list", element: <ComingSoonStub title="My Reading List" description="Browse your assigned reading list." backTo="/student" backLabel="Back to Dashboard" /> },
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
              { path: "flags", element: <FeatureFlags /> },
              { path: "audit", element: <AuditLog /> },
              { path: "methodologies", element: <MethodologyConfigPage /> },
              // Admin stubs
              { path: "content-flags", element: <ComingSoonStub title="Content Flags" description="Review flagged content across the platform." backTo="/admin" backLabel="Back to Admin" /> },
              { path: "appeals", element: <ComingSoonStub title="Appeals" description="Review user moderation appeals." backTo="/admin" backLabel="Back to Admin" /> },
              { path: "reports", element: <ComingSoonStub title="Reports" description="View platform analytics and usage reports." backTo="/admin" backLabel="Back to Admin" /> },
              { path: "system", element: <ComingSoonStub title="System Configuration" description="System health monitoring and configuration." backTo="/admin" backLabel="Back to Admin" /> },
              // Admin redirects
              { path: "feature-flags", element: <Navigate to="/admin/flags" replace /> },
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
          { index: true, element: <ComplianceSetup />, errorElement: <RouteErrorBoundary /> },
          { path: "attendance", element: <AttendanceTracker /> },
          { path: "assessments", element: <AssessmentRecords /> },
          { path: "tests", element: <StandardizedTests /> },
          { path: "portfolios", element: <PortfolioList /> },
          { path: "portfolios/:studentId/:id", element: <PortfolioBuilder /> },
          { path: "transcripts", element: <TranscriptList /> },
          { path: "transcripts/:studentId/:id", element: <TranscriptBuilder /> },
          // Compliance stubs
          { path: "portfolios/new", element: <ComingSoonStub title="New Portfolio" description="Create a new student portfolio for compliance records." backTo="/compliance/portfolios" backLabel="Back to Portfolios" /> },
          { path: "portfolios/:id", element: <ComingSoonStub title="Portfolio" description="View portfolio details and artifacts." backTo="/compliance/portfolios" backLabel="Back to Portfolios" /> },
          { path: "immunization", element: <ComingSoonStub title="Immunization Records" description="Track immunization records for compliance." backTo="/compliance" backLabel="Back to Compliance" /> },
          { path: "submissions", element: <ComingSoonStub title="Submissions" description="View and manage compliance document submissions." backTo="/compliance" backLabel="Back to Compliance" /> },
          { path: "requirements", element: <ComingSoonStub title="Requirements" description="View state-specific homeschooling requirements." backTo="/compliance" backLabel="Back to Compliance" /> },
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
