import { useState } from "react";
import { Search, Home, Settings, Bell, BookOpen } from "lucide-react";
import {
  Button,
  Input,
  Textarea,
  Select,
  Checkbox,
  Radio,
  FormField,
  Card,
  Badge,
  Avatar,
  Icon,
  Spinner,
  Skeleton,
  Tooltip,
  Tabs,
  ProgressBar,
  EmptyState,
  StatCard,
  Breadcrumb,
  StarRating,
  DatePicker,
  Link,
  List,
  ListItem,
} from "./components/ui";
import { SkipLink, PageTitle, MethodologyBadge, NetworkStatus, TierGate } from "./components/common";
import { ToastProvider, useToast } from "./components/ui";

function ToastDemo() {
  const { toast } = useToast();
  return (
    <div className="flex gap-2 flex-wrap">
      <Button size="sm" variant="primary" onClick={() => toast("Item saved successfully!", "success")}>
        Success toast
      </Button>
      <Button size="sm" variant="secondary" onClick={() => toast("Something went wrong.", "error")}>
        Error toast
      </Button>
      <Button size="sm" variant="tertiary" onClick={() => toast("Check your settings.", "warning")}>
        Warning toast
      </Button>
    </div>
  );
}

export function App() {
  const [inputValue, setInputValue] = useState("");
  const [rating, setRating] = useState(3);
  const [selectedDate, setSelectedDate] = useState("");

  return (
    <ToastProvider>
      <SkipLink />
      <NetworkStatus />
      <div className="min-h-screen bg-surface" data-context="parent">
        <div className="max-w-5xl mx-auto p-spacing-page-x py-8">
          <PageTitle title="Component Library" subtitle="Phase 3 — Shared UI Component Library" />

          <div className="mt-8 flex flex-col gap-spacing-section-gap" id="main-content">

            {/* Buttons */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Buttons</h2>
              <div className="flex flex-wrap gap-3 items-center">
                <Button variant="primary">Primary</Button>
                <Button variant="secondary">Secondary</Button>
                <Button variant="tertiary">Tertiary</Button>
                <Button variant="gradient">Gradient CTA</Button>
                <Button variant="primary" loading>Loading</Button>
                <Button variant="primary" disabled>Disabled</Button>
                <Button variant="primary" size="sm">Small</Button>
                <Button variant="primary" size="lg">Large</Button>
                <Button variant="primary" leadingIcon={<Icon icon={Search} size="sm" />}>
                  With Icon
                </Button>
              </div>
            </section>

            {/* Form Elements */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Form Elements</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-2xl">
                <FormField label="Email" required>
                  {({ id }) => (
                    <Input
                      id={id}
                      type="email"
                      placeholder="name@example.com"
                      value={inputValue}
                      onChange={(e) => setInputValue(e.target.value)}
                    />
                  )}
                </FormField>

                <FormField label="With Error" error="This field is required">
                  {({ id }) => (
                    <Input id={id} error placeholder="Error state" />
                  )}
                </FormField>

                <FormField label="Country">
                  {({ id }) => (
                    <Select id={id}>
                      <option value="">Choose one</option>
                      <option value="us">United States</option>
                      <option value="ca">Canada</option>
                      <option value="uk">United Kingdom</option>
                    </Select>
                  )}
                </FormField>

                <FormField label="Message" hint="Max 500 characters">
                  {({ id }) => (
                    <Textarea id={id} placeholder="Type your message..." />
                  )}
                </FormField>

                <div className="flex flex-col gap-3">
                  <Checkbox label="I agree to the terms" />
                  <Checkbox label="Send me updates" defaultChecked />
                </div>

                <div className="flex flex-col gap-3">
                  <Radio name="plan" label="Free plan" value="free" defaultChecked />
                  <Radio name="plan" label="Premium plan" value="premium" />
                </div>
              </div>
            </section>

            {/* Cards & Content */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Cards</h2>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Card>
                  <p className="type-title-md text-on-surface">Standard Card</p>
                  <p className="type-body-md text-on-surface-variant mt-2">Non-interactive card with tonal depth.</p>
                </Card>
                <Card interactive>
                  <p className="type-title-md text-on-surface">Interactive Card</p>
                  <p className="type-body-md text-on-surface-variant mt-2">Hover for shadow lift effect.</p>
                </Card>
                <StatCard
                  value={42}
                  label="Activities This Week"
                  trend={{ direction: "up", label: "+12% from last week" }}
                  icon={<Icon icon={BookOpen} size="lg" />}
                />
              </div>
            </section>

            {/* Badges & Avatars */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Badges & Avatars</h2>
              <div className="flex flex-wrap items-center gap-3">
                <Badge>Default</Badge>
                <Badge variant="primary">Primary</Badge>
                <Badge variant="secondary">Secondary</Badge>
                <Badge variant="success">Success</Badge>
                <Badge variant="warning">Warning</Badge>
                <Badge variant="error">Error</Badge>
                <MethodologyBadge slug="charlotte-mason" label="Charlotte Mason" />
              </div>
              <div className="flex items-center gap-3 mt-4">
                <Avatar name="Jane Doe" size="xs" />
                <Avatar name="Jane Doe" size="sm" />
                <Avatar name="Jane Doe" size="md" />
                <Avatar name="Jane Doe" size="lg" />
                <Avatar name="Jane Doe" size="xl" />
              </div>
            </section>

            {/* Breadcrumb */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Breadcrumb</h2>
              <Breadcrumb
                items={[
                  { label: "Home", href: "/" },
                  { label: "Settings", href: "/settings" },
                  { label: "Account" },
                ]}
              />
            </section>

            {/* Tabs */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Tabs</h2>
              <Tabs
                tabs={[
                  { id: "overview", label: "Overview", content: <p className="type-body-md text-on-surface-variant">Overview content here.</p> },
                  { id: "activity", label: "Activity", content: <p className="type-body-md text-on-surface-variant">Activity log here.</p> },
                  { id: "settings", label: "Settings", content: <p className="type-body-md text-on-surface-variant">Settings panel here.</p> },
                ]}
                className="max-w-lg"
              />
            </section>

            {/* Progress & Rating */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Progress & Rating</h2>
              <div className="flex flex-col gap-4 max-w-md">
                <div>
                  <p className="type-label-md text-on-surface-variant mb-1">Progress (75%)</p>
                  <ProgressBar value={75} />
                </div>
                <div>
                  <p className="type-label-md text-on-surface-variant mb-1">Star Rating</p>
                  <StarRating value={rating} onChange={setRating} />
                </div>
                <div>
                  <p className="type-label-md text-on-surface-variant mb-1">Read-only Rating</p>
                  <StarRating value={4} readOnly />
                </div>
              </div>
            </section>

            {/* Date Picker */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Date Picker</h2>
              <DatePicker
                value={selectedDate}
                onChange={setSelectedDate}
              />
              {selectedDate && (
                <p className="type-body-md text-on-surface-variant mt-2">
                  Selected: {selectedDate}
                </p>
              )}
            </section>

            {/* Loading States */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Loading States</h2>
              <div className="flex items-center gap-4">
                <Spinner size="sm" className="text-primary" />
                <Spinner size="md" className="text-primary" />
                <Spinner size="lg" className="text-primary" />
              </div>
              <div className="flex flex-col gap-2 mt-4 max-w-sm">
                <Skeleton width="w-3/4" height="h-4" />
                <Skeleton width="w-full" height="h-4" />
                <Skeleton width="w-1/2" height="h-4" />
                <Skeleton width="w-10" height="h-10" rounded />
              </div>
            </section>

            {/* Tooltip & Icon */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Tooltips & Icons</h2>
              <div className="flex items-center gap-4">
                <Tooltip content="Home page">
                  <Button variant="tertiary" size="sm">
                    <Icon icon={Home} size="md" />
                  </Button>
                </Tooltip>
                <Tooltip content="Settings">
                  <Button variant="tertiary" size="sm">
                    <Icon icon={Settings} size="md" />
                  </Button>
                </Tooltip>
                <Tooltip content="Notifications">
                  <Button variant="tertiary" size="sm">
                    <Icon icon={Bell} size="md" />
                  </Button>
                </Tooltip>
              </div>
              <div className="flex items-center gap-3 mt-4">
                <Icon icon={Home} size="xs" className="text-on-surface-variant" />
                <Icon icon={Home} size="sm" className="text-on-surface-variant" />
                <Icon icon={Home} size="md" className="text-on-surface-variant" />
                <Icon icon={Home} size="lg" className="text-on-surface-variant" />
                <Icon icon={Home} size="xl" className="text-on-surface-variant" />
                <Icon icon={Home} size="2xl" className="text-on-surface-variant" />
              </div>
            </section>

            {/* List */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">List</h2>
              <div className="max-w-md">
                <List>
                  <ListItem>
                    <Card>
                      <p className="type-body-md text-on-surface">First item — spaced with list gap</p>
                    </Card>
                  </ListItem>
                  <ListItem>
                    <Card>
                      <p className="type-body-md text-on-surface">Second item</p>
                    </Card>
                  </ListItem>
                  <ListItem>
                    <Card>
                      <p className="type-body-md text-on-surface">Third item</p>
                    </Card>
                  </ListItem>
                </List>
              </div>
            </section>

            {/* Link */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Links</h2>
              <div className="flex gap-4">
                <Link href="#">Internal link</Link>
                <Link href="https://example.com" external>External link</Link>
              </div>
            </section>

            {/* Toast */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Toasts</h2>
              <ToastDemo />
            </section>

            {/* Empty State */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Empty State</h2>
              <EmptyState
                message="No activities yet"
                description="Start by adding your first learning activity. Activities help track your student's progress."
                illustration={<Icon icon={BookOpen} size="2xl" />}
                action={<Button variant="primary">Add Activity</Button>}
              />
            </section>

            {/* Tier Gate */}
            <section>
              <h2 className="type-headline-sm text-on-surface mb-4">Tier Gate</h2>
              <TierGate featureName="Advanced Analytics" />
            </section>

          </div>
        </div>
      </div>
    </ToastProvider>
  );
}
