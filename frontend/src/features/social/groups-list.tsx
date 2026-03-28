import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { Users, Globe, Lock, UserPlus } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Tabs,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useMyGroups,
  usePlatformGroups,
  useDiscoverGroups,
  useJoinGroup,
} from "@/hooks/use-social";
import type { GroupDetailResponse, GroupSummaryResponse } from "@/hooks/use-social";

// ─── Join policy icon ───────────────────────────────────────────────────────

function JoinPolicyIcon({ policy }: { policy: string }) {
  if (policy === "open") return <Icon icon={Globe} size="xs" />;
  if (policy === "invite_only") return <Icon icon={Lock} size="xs" />;
  return <Icon icon={UserPlus} size="xs" />;
}

// ─── Group card ─────────────────────────────────────────────────────────────

function GroupCard({
  group,
  showJoin,
}: {
  group: GroupDetailResponse | GroupSummaryResponse;
  showJoin?: boolean;
}) {
  const joinGroup = useJoinGroup();

  const summary = "summary" in group ? group.summary : group;
  const myStatus = "my_status" in group ? group.my_status : undefined;

  return (
    <Card className="p-card-padding">
      <RouterLink
        to={`/groups/${summary.id}`}
        className="block hover:opacity-90 transition-opacity"
      >
        <div className="flex items-start gap-3">
          <div className="w-12 h-12 rounded-radius-md bg-secondary-container flex items-center justify-center shrink-0">
            <Icon icon={Users} size="lg" className="text-on-secondary-container" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="type-title-sm text-on-surface">{summary.name}</p>
            {summary.description && (
              <p className="type-body-sm text-on-surface-variant line-clamp-2 mt-0.5">
                {summary.description}
              </p>
            )}
            <div className="flex items-center gap-3 mt-2">
              <span className="type-label-sm text-on-surface-variant flex items-center gap-1">
                <Icon icon={Users} size="xs" />
                <FormattedMessage
                  id="social.groups.members"
                  values={{ count: summary.member_count }}
                />
              </span>
              {summary.methodology_name && (
                <Badge variant="secondary">
                  {summary.methodology_name}
                </Badge>
              )}
              <span className="type-label-sm text-on-surface-variant flex items-center gap-1">
                <JoinPolicyIcon policy={summary.join_policy} />
                {summary.join_policy.replace("_", " ")}
              </span>
            </div>
          </div>
        </div>
      </RouterLink>

      {showJoin && !summary.is_member && (
        <div className="mt-3 pt-3 border-t border-outline-variant/10">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => joinGroup.mutate(summary.id)}
            disabled={joinGroup.isPending}
          >
            <FormattedMessage id="social.groups.join" />
          </Button>
        </div>
      )}

      {summary.is_member && myStatus === "pending" && (
        <div className="mt-3 pt-3 border-t border-outline-variant/10">
          <Badge variant="secondary">
            <FormattedMessage id="social.groups.pending" />
          </Badge>
        </div>
      )}
    </Card>
  );
}

// ─── Groups list page ───────────────────────────────────────────────────────

export function GroupsList() {
  const intl = useIntl();
  const { data: myGroups, isPending: myGroupsPending } = useMyGroups();
  const { data: platformGroups, isPending: platformPending } = usePlatformGroups();
  const { data: discoverGroups } = useDiscoverGroups();

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={intl.formatMessage({ id: "social.groups.title" })} />

      <Tabs
        defaultTab="mine"
        tabs={[
          {
            id: "mine",
            label: `${intl.formatMessage({ id: "social.groups.tabs.mine" })} (${myGroups?.length ?? 0})`,
            content: (
              <div className="mt-6 space-y-4">
                {myGroupsPending && (
                  <div className="space-y-3">
                    {[1, 2, 3].map((n) => (
                      <Skeleton key={n} className="h-24 w-full rounded-radius-md" />
                    ))}
                  </div>
                )}

                {myGroups && myGroups.length === 0 && (
                  <EmptyState
                    illustration={<Icon icon={Users} size="xl" />}
                    message={intl.formatMessage({ id: "social.groups.empty.title" })}
                    description={intl.formatMessage({ id: "social.groups.empty.description" })}
                  />
                )}

                {myGroups?.map((group) => (
                  <GroupCard key={group.summary.id} group={group} />
                ))}
              </div>
            ),
          },
          {
            id: "discover",
            label: intl.formatMessage({ id: "social.groups.tabs.discover" }),
            content: (
              <div className="mt-6 space-y-6">
                {/* Platform groups */}
                {platformGroups && platformGroups.length > 0 && (
                  <div className="space-y-3">
                    <h3 className="type-title-md text-on-surface">Platform Groups</h3>
                    {platformGroups.map((group) => (
                      <GroupCard key={group.summary.id} group={group} showJoin />
                    ))}
                  </div>
                )}

                {/* Discovered groups */}
                {discoverGroups && discoverGroups.length > 0 && (
                  <div className="space-y-3">
                    <h3 className="type-title-md text-on-surface">Suggested Groups</h3>
                    {discoverGroups.map((group) => (
                      <GroupCard key={group.id} group={group} showJoin />
                    ))}
                  </div>
                )}

                {platformPending && (
                  <div className="space-y-3">
                    {[1, 2].map((n) => (
                      <Skeleton key={n} className="h-24 w-full rounded-radius-md" />
                    ))}
                  </div>
                )}
              </div>
            ),
          },
        ]}
      />
    </div>
  );
}
