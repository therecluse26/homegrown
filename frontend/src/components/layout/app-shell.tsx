import { type ReactNode, Suspense, useEffect, useRef, useState } from "react";
import { NavLink, Outlet, useLocation, Link, useNavigate } from "react-router";
import {
  Home,
  BookOpen,
  Users,
  ShoppingBag,
  Calendar,
  ClipboardList,
  Settings,
  MessageSquare,
  Plus,
  MoreHorizontal,
  LogOut,
  User,
  CreditCard,
  Star,
  Search,
  HelpCircle,
} from "lucide-react";
import { useIntl } from "react-intl";
import { Icon, Spinner } from "@/components/ui";
import { DropdownMenu, DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { SkipLink } from "@/components/common";
import { useAuthContext } from "@/features/auth/auth-provider";
import { NotificationBell } from "@/components/layout/notification-bell";
import { CartBadge } from "@/components/layout/cart-badge";
import { SearchBar } from "@/components/layout/search-bar";
import { KeyboardShortcutsModal } from "@/components/layout/keyboard-shortcuts-modal";
import { useWebSocket } from "@/hooks/use-websocket";
import {
  KeyboardShortcutRegistryProvider,
  useKeyboardShortcutRegistry,
} from "@/hooks/use-keyboard-shortcut-registry";
import { CoppaReverificationBanner } from "@/features/auth/coppa-reverification-banner";
import { initLogout, performLogout } from "@/lib/kratos";

type NavItem = {
  to: string;
  icon: typeof Home;
  labelId: string;
  end?: boolean;
};

const navItems: NavItem[] = [
  { to: "/", icon: Home, labelId: "nav.home", end: true },
  { to: "/learning", icon: BookOpen, labelId: "nav.learning" },
  { to: "/recommendations", icon: Star, labelId: "nav.recommendations" },
  { to: "/friends", icon: Users, labelId: "nav.community" },
  { to: "/calendar", icon: Calendar, labelId: "nav.calendar" },
  { to: "/marketplace", icon: ShoppingBag, labelId: "nav.marketplace" },
  { to: "/compliance", icon: ClipboardList, labelId: "nav.compliance" },
  { to: "/settings", icon: Settings, labelId: "nav.settings" },
];

function useLogout() {
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const handleLogout = async () => {
    if (isLoggingOut) return;
    setIsLoggingOut(true);
    try {
      const { logout_token } = await initLogout();
      await performLogout(logout_token);
      window.location.href = "/auth/login";
    } catch {
      window.location.href = "/auth/login";
    }
  };
  return { handleLogout, isLoggingOut };
}

function SidebarNav() {
  const intl = useIntl();

  return (
    <nav
      aria-label={intl.formatMessage({ id: "nav.landmark.main", defaultMessage: "Main navigation" })}
      className="hidden lg:flex flex-col fixed top-0 left-0 h-full bg-surface-container-low/80 backdrop-blur-[20px] z-[var(--z-sticky)]"
      style={{ width: "var(--width-sidebar)" }}
    >
      <div className="p-card-padding">
        <p className="type-title-md text-primary font-semibold">
          {intl.formatMessage({ id: "app.name", defaultMessage: "Homegrown Academy" })}
        </p>
      </div>
      <ul className="flex flex-col gap-1 px-3 flex-1">
        {navItems.map((item) => (
          <li key={item.to}>
            <NavLink
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-button type-label-lg text-on-surface-variant transition-colors duration-[var(--duration-normal)] ${
                  isActive
                    ? "bg-primary/10 text-primary font-semibold"
                    : "hover:bg-surface-container-high"
                }`
              }
            >
              <Icon icon={item.icon} size="md" />
              <span>{intl.formatMessage({ id: item.labelId })}</span>
            </NavLink>
          </li>
        ))}
      </ul>
      <div className="px-3 pb-4 flex flex-col gap-1">
        <NavLink
          to="/help"
          className={({ isActive }) =>
            `flex items-center gap-3 px-3 py-2.5 rounded-button type-label-lg text-on-surface-variant transition-colors duration-[var(--duration-normal)] ${
              isActive
                ? "bg-primary/10 text-primary font-semibold"
                : "hover:bg-surface-container-high"
            }`
          }
        >
          <Icon icon={HelpCircle} size="md" />
          <span>{intl.formatMessage({ id: "footer.help", defaultMessage: "Help & Support" })}</span>
        </NavLink>
      </div>
    </nav>
  );
}

type BottomMoreSheetProps = { onClose: () => void };

function BottomMoreSheet({ onClose }: BottomMoreSheetProps) {
  const intl = useIntl();
  const navigate = useNavigate();
  const { handleLogout } = useLogout();

  const moreItems = [
    { to: "/calendar", icon: Calendar, labelId: "nav.calendar" },
    { to: "/compliance", icon: ClipboardList, labelId: "nav.compliance" },
    { to: "/messages", icon: MessageSquare, labelId: "nav.messages" },
    { to: "/settings", icon: Settings, labelId: "nav.settings" },
  ];

  function handleNav(to: string) {
    navigate(to);
    onClose();
  }

  return (
    <>
      <div
        className="fixed inset-0 z-[var(--z-overlay)] bg-scrim/40"
        onClick={onClose}
        aria-hidden
      />
      <div
        role="dialog"
        aria-label={intl.formatMessage({ id: "nav.more", defaultMessage: "More" })}
        className="fixed bottom-16 left-0 right-0 z-[var(--z-modal)] bg-surface-container-lowest rounded-t-2xl shadow-ambient-lg safe-area-pb"
      >
        <div className="flex flex-col py-2">
          {moreItems.map((item) => (
            <button
              key={item.to}
              onClick={() => handleNav(item.to)}
              className="flex items-center gap-3 px-6 py-3 type-label-lg text-on-surface-variant hover:bg-surface-container-low transition-colors"
            >
              <Icon icon={item.icon} size="md" />
              <span>{intl.formatMessage({ id: item.labelId })}</span>
            </button>
          ))}
          <div className="border-t border-outline-variant my-1 mx-4" />
          <button
            onClick={() => { void handleLogout(); onClose(); }}
            className="flex items-center gap-3 px-6 py-3 type-label-lg text-error hover:bg-surface-container-low transition-colors"
          >
            <Icon icon={LogOut} size="md" />
            <span>{intl.formatMessage({ id: "nav.logout", defaultMessage: "Log out" })}</span>
          </button>
        </div>
      </div>
    </>
  );
}

function BottomNav() {
  const intl = useIntl();
  const [moreOpen, setMoreOpen] = useState(false);
  const primaryItems = navItems.slice(0, 4);

  return (
    <>
      {moreOpen && <BottomMoreSheet onClose={() => setMoreOpen(false)} />}
      <nav
        aria-label={intl.formatMessage({ id: "nav.landmark.mobile", defaultMessage: "Mobile navigation" })}
        className="lg:hidden fixed bottom-0 left-0 right-0 bg-surface-container-low/80 backdrop-blur-[20px] z-[var(--z-sticky)] safe-area-pb"
      >
        <ul className="flex justify-around items-center h-16">
          {primaryItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  `flex flex-col items-center gap-0.5 px-3 py-1.5 min-w-[3rem] rounded-button transition-colors duration-[var(--duration-normal)] ${
                    isActive ? "text-primary" : "text-on-surface-variant"
                  }`
                }
              >
                <Icon icon={item.icon} size="md" />
                <span className="type-label-sm">
                  {intl.formatMessage({ id: item.labelId })}
                </span>
              </NavLink>
            </li>
          ))}
          <li>
            <button
              onClick={() => setMoreOpen(true)}
              aria-label={intl.formatMessage({ id: "nav.more", defaultMessage: "More" })}
              aria-expanded={moreOpen}
              className="flex flex-col items-center gap-0.5 px-3 py-1.5 min-w-[3rem] rounded-button text-on-surface-variant transition-colors duration-[var(--duration-normal)] hover:text-primary"
            >
              <Icon icon={MoreHorizontal} size="md" />
              <span className="type-label-sm">
                {intl.formatMessage({ id: "nav.more", defaultMessage: "More" })}
              </span>
            </button>
          </li>
        </ul>
      </nav>
    </>
  );
}

function Header() {
  const intl = useIntl();
  const { user } = useAuthContext();
  const { handleLogout } = useLogout();
  const navigate = useNavigate();

  const initials = user?.display_name?.[0]?.toUpperCase() ?? "?";
  const familyName = user?.family_display_name ?? user?.display_name ?? "";

  return (
    <header className="flex items-center justify-between py-2 lg:py-3">
      <div className="lg:hidden">
        <p className="type-title-md text-primary font-semibold">
          {intl.formatMessage({ id: "app.name", defaultMessage: "Homegrown Academy" })}
        </p>
      </div>
      <div className="flex items-center gap-3 ml-auto">
        <SearchBar />
        <NavLink
          to="/search"
          className="md:hidden p-2 min-w-11 min-h-11 flex items-center justify-center rounded-button text-on-surface-variant hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
          aria-label={intl.formatMessage({ id: "nav.search", defaultMessage: "Search" })}
        >
          <Icon icon={Search} size="md" />
        </NavLink>
        <CartBadge />
        <NotificationBell />

        {/* Messages */}
        <NavLink
          to="/messages"
          className="p-2 min-w-11 min-h-11 flex items-center justify-center rounded-button text-on-surface-variant hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
          aria-label={intl.formatMessage({ id: "nav.messages", defaultMessage: "Messages" })}
        >
          <Icon icon={MessageSquare} size="md" />
        </NavLink>

        {/* Create (+) */}
        <DropdownMenu
          trigger={
            <button
              aria-label={intl.formatMessage({ id: "nav.create", defaultMessage: "Create" })}
              className="bg-primary text-on-primary rounded-full w-9 h-9 flex items-center justify-center hover:bg-primary/90 transition-colors"
            >
              <Icon icon={Plus} size="md" />
            </button>
          }
        >
          <DropdownMenuItem
            onClick={() => document.dispatchEvent(new CustomEvent("open:post-composer"))}
          >
            <Icon icon={Plus} size="sm" />
            {intl.formatMessage({ id: "nav.create.post", defaultMessage: "New Post" })}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate("/events/new")}>
            <Icon icon={Calendar} size="sm" />
            {intl.formatMessage({ id: "nav.create.event", defaultMessage: "New Event" })}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate("/messages")}>
            <Icon icon={MessageSquare} size="sm" />
            {intl.formatMessage({ id: "nav.create.message", defaultMessage: "New Message" })}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate("/creator/listings/new")}>
            <Icon icon={ShoppingBag} size="sm" />
            {intl.formatMessage({ id: "nav.create.listing", defaultMessage: "New Listing" })}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate("/learning/activities/new")}>
            <Icon icon={BookOpen} size="sm" />
            {intl.formatMessage({ id: "nav.create.activity", defaultMessage: "Log Activity" })}
          </DropdownMenuItem>
        </DropdownMenu>

        {/* User menu */}
        <DropdownMenu
          trigger={
            <button className="flex items-center gap-2">
              <span className="bg-primary-container text-on-primary-container rounded-full w-8 h-8 flex items-center justify-center type-label-md font-semibold">
                {initials}
              </span>
              <span className="hidden sm:block type-label-md text-on-surface-variant truncate max-w-[8rem]">
                {familyName}
              </span>
            </button>
          }
        >
          <DropdownMenuItem onClick={() => navigate("/profile")}>
            <Icon icon={User} size="sm" />
            {intl.formatMessage({ id: "nav.profile", defaultMessage: "My Profile" })}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate("/settings/account")}>
            <Icon icon={Settings} size="sm" />
            {intl.formatMessage({ id: "nav.account-settings", defaultMessage: "Account Settings" })}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate("/settings")}>
            <Icon icon={Home} size="sm" />
            {intl.formatMessage({ id: "nav.family-settings", defaultMessage: "Family Settings" })}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate("/billing")}>
            <Icon icon={CreditCard} size="sm" />
            {intl.formatMessage({ id: "nav.billing", defaultMessage: "Billing & Subscription" })}
          </DropdownMenuItem>
          <div className="border-t border-outline-variant my-1" />
          <DropdownMenuItem destructive onClick={() => void handleLogout()}>
            <Icon icon={LogOut} size="sm" />
            {intl.formatMessage({ id: "nav.logout", defaultMessage: "Log out" })}
          </DropdownMenuItem>
        </DropdownMenu>
      </div>
    </header>
  );
}

function AppFooter() {
  const intl = useIntl();
  return (
    <footer
      aria-label={intl.formatMessage({ id: "footer.landmark", defaultMessage: "Site footer" })}
      className="no-print mt-8 py-5 bg-surface-container-low"
    >
      <nav
        aria-label={intl.formatMessage({ id: "footer.nav", defaultMessage: "Legal and help links" })}
        className="flex flex-wrap justify-center gap-x-6 gap-y-2"
      >
        <Link
          to="/legal/privacy"
          className="type-label-sm text-on-surface-variant hover:text-primary transition-colors duration-[var(--duration-normal)] underline-offset-4 hover:underline"
        >
          {intl.formatMessage({ id: "legal.privacy.title", defaultMessage: "Privacy Policy" })}
        </Link>
        <Link
          to="/legal/terms"
          className="type-label-sm text-on-surface-variant hover:text-primary transition-colors duration-[var(--duration-normal)] underline-offset-4 hover:underline"
        >
          {intl.formatMessage({ id: "legal.terms.title", defaultMessage: "Terms of Service" })}
        </Link>
        <Link
          to="/legal/guidelines"
          className="type-label-sm text-on-surface-variant hover:text-primary transition-colors duration-[var(--duration-normal)] underline-offset-4 hover:underline"
        >
          {intl.formatMessage({ id: "legal.guidelines.title", defaultMessage: "Community Guidelines" })}
        </Link>
        <Link
          to="/help"
          className="type-label-sm text-on-surface-variant hover:text-primary transition-colors duration-[var(--duration-normal)] underline-offset-4 hover:underline"
        >
          {intl.formatMessage({ id: "footer.help", defaultMessage: "Help & Support" })}
        </Link>
      </nav>
      <p className="mt-3 text-center type-label-sm text-on-surface-variant/60">
        {intl.formatMessage(
          { id: "footer.copyright", defaultMessage: "© {year} Homegrown Academy" },
          { year: new Date().getFullYear() },
        )}
      </p>
    </footer>
  );
}

function GlobalShortcutHandler() {
  const { isOpen, toggleShortcuts, closeShortcuts } = useKeyboardShortcutRegistry();
  // Use a ref so the stable event listener always reads current isOpen
  const isOpenRef = useRef(isOpen);
  isOpenRef.current = isOpen;

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Escape closes the shortcuts modal regardless of what has focus
      if (e.key === "Escape" && isOpenRef.current) {
        closeShortcuts();
        return;
      }

      const target = e.target as HTMLElement;
      if (
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable
      ) return;

      if (e.key === "?") {
        e.preventDefault();
        toggleShortcuts();
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [toggleShortcuts, closeShortcuts]);

  return null;
}

function AppShellInner({ children }: { children?: ReactNode }) {
  const location = useLocation();

  useWebSocket();

  return (
    <>
      <GlobalShortcutHandler />
      <KeyboardShortcutsModal />
      <SkipLink />
      <SidebarNav />
      <BottomNav />
      <div
        className="min-h-screen bg-surface lg:pl-[var(--width-sidebar)]"
        data-context="parent"
      >
        <div className="sticky top-0 z-[var(--z-sticky)] bg-surface/80 backdrop-blur-[20px]">
          <div className="max-w-[var(--width-content)] mx-auto px-spacing-page-x lg:px-spacing-page-x-lg">
            <Header />
          </div>
        </div>
        <div className="max-w-[var(--width-content)] mx-auto px-spacing-page-x lg:px-spacing-page-x-lg">
          <CoppaReverificationBanner />
          <main id="main-content" key={location.pathname}>
            <Suspense
              fallback={
                <div className="flex items-center justify-center py-12">
                  <Spinner size="lg" className="text-primary" />
                </div>
              }
            >
              {children ?? <Outlet />}
            </Suspense>
          </main>
          <AppFooter />
          {/* Bottom clearance for mobile BottomNav */}
          <div className="h-16 lg:h-0 safe-area-pb" />
        </div>
      </div>
    </>
  );
}

export function AppShell({ children }: { children?: ReactNode }) {
  return (
    <KeyboardShortcutRegistryProvider>
      <AppShellInner>{children}</AppShellInner>
    </KeyboardShortcutRegistryProvider>
  );
}
