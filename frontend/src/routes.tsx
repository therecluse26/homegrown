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
const JournalDetail = lazy(() => import("@/features/learning/journal-detail").then(m => ({ default: m.JournalDetail })));
const ActivityNew = lazy(() => import("@/features/learning/activity-new").then(m => ({ default: m.ActivityNew })));
const ActivityDetail = lazy(() => import("@/features/learning/activity-detail").then(m => ({ default: m.ActivityDetail })));
const GradeNew = lazy(() => import("@/features/learning/grade-new").then(m => ({ default: m.GradeNew })));
const GradeDetail = lazy(() => import("@/features/learning/grade-detail").then(m => ({ default: m.GradeDetail })));
const ReadingListDetail = lazy(() => import("@/features/learning/reading-list-detail").then(m => ({ default: m.ReadingListDetail })));
const ReadingListBooks = lazy(() => import("@/features/learning/reading-list-books").then(m => ({ default: m.ReadingListBooks })));
const NatureJournalDetail = lazy(() => import("@/features/learning/nature-journal-detail").then(m => ({ default: m.NatureJournalDetail })));
const TriviumDetail = lazy(() => import("@/features/learning/trivium-detail").then(m => ({ default: m.TriviumDetail })));

// Student
const StudentDashboardPage = lazy(() => import("@/features/student/student-dashboard").then(m => ({ default: m.StudentDashboard })));
const StudentQuiz = lazy(() => import("@/features/student/student-quiz").then(m => ({ default: m.StudentQuiz })));
const StudentVideo = lazy(() => import("@/features/student/student-video").then(m => ({ default: m.StudentVideo })));
const StudentReader = lazy(() => import("@/features/student/student-reader").then(m => ({ default: m.StudentReader })));
const StudentSequence = lazy(() => import("@/features/student/student-sequence").then(m => ({ default: m.StudentSequence })));
const StudentJournal = lazy(() => import("@/features/student/student-journal").then(m => ({ default: m.StudentJournal })));
const StudentActivities = lazy(() => import("@/features/student/student-activities").then(m => ({ default: m.StudentActivities })));
const StudentReadingList = lazy(() => import("@/features/student/student-reading-list").then(m => ({ default: m.StudentReadingList })));
const StudentLogin = lazy(() => import("@/features/student/student-login").then(m => ({ default: m.StudentLogin })));

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
const MyLibrary = lazy(() => import("@/features/marketplace/my-library").then(m => ({ default: m.MyLibrary })));
const CategoryBrowse = lazy(() => import("@/features/marketplace/category-browse").then(m => ({ default: m.CategoryBrowse })));
const CreatorEarnings = lazy(() => import("@/features/marketplace/creator/creator-earnings").then(m => ({ default: m.CreatorEarnings })));
const CreatorAnalytics = lazy(() => import("@/features/marketplace/creator/creator-analytics").then(m => ({ default: m.CreatorAnalytics })));

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
const MicroChargeHistory = lazy(() => import("@/features/billing/micro-charge-history").then(m => ({ default: m.MicroChargeHistory })));

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
const AdminAppeals = lazy(() => import("@/features/admin/appeals").then(m => ({ default: m.Appeals })));
const AdminContentFlags = lazy(() => import("@/features/admin/content-flags").then(m => ({ default: m.ContentFlags })));
const AdminSafetyReports = lazy(() => import("@/features/admin/safety-reports").then(m => ({ default: m.SafetyReports })));
const AdminSystemDashboard = lazy(() => import("@/features/admin/system-dashboard").then(m => ({ default: m.SystemDashboard })));

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
const PortfolioNew = lazy(() => import("@/features/compliance/portfolio-new").then(m => ({ default: m.PortfolioNew })));
const ComplianceRequirements = lazy(() => import("@/features/compliance/requirements").then(m => ({ default: m.Requirements })));

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

              // ─── Learning detail pages ────────────────────────────────────
              { path: "learning/journals/:id", element: <JournalDetail /> },
              { path: "learning/nature-journal/:id", element: <NatureJournalDetail /> },
              { path: "learning/trivium-tracker/:id", element: <TriviumDetail /> },
              { path: "learning/grades/new", element: <GradeNew /> },
              { path: "learning/grades/:id", element: <GradeDetail /> },
              { path: "learning/reading-lists/:id/books", element: <ReadingListBooks /> },
              { path: "learning/reading-lists/:id", element: <ReadingListDetail /> },
              { path: "learning/activities/new", element: <ActivityNew /> },
              { path: "learning/activities/:id", element: <ActivityDetail /> },
              // Marketplace pages
              { path: "marketplace/library", element: <MyLibrary /> },
              { path: "marketplace/categories", element: <CategoryBrowse /> },
              // Creator pages
              { path: "creator/earnings", element: <CreatorEarnings /> },
              { path: "creator/analytics", element: <CreatorAnalytics /> },
              // Billing pages
              { path: "billing/micro-charges", element: <MicroChargeHistory /> },

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
      { path: "student/login", element: <StudentLogin /> },
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
              // Student views
              { path: "journal", element: <StudentJournal /> },
              { path: "activities", element: <StudentActivities /> },
              { path: "reading-list", element: <StudentReadingList /> },
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
              // Admin pages
              { path: "content-flags", element: <AdminContentFlags /> },
              { path: "appeals", element: <AdminAppeals /> },
              { path: "reports", element: <AdminSafetyReports /> },
              { path: "system", element: <AdminSystemDashboard /> },
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
          // Compliance pages
          { path: "portfolios/new", element: <PortfolioNew /> },
          { path: "requirements", element: <ComplianceRequirements /> },
          // Deferred — need new backend models
          { path: "immunization", element: <ComingSoonStub title="Immunization Records" description="Track immunization records for compliance." backTo="/compliance" backLabel="Back to Compliance" /> },
          { path: "submissions", element: <ComingSoonStub title="Submissions" description="View and manage compliance document submissions." backTo="/compliance" backLabel="Back to Compliance" /> },
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
