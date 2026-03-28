import { useState } from "react";
import { useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { Search, Users } from "lucide-react";
import {
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Input,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useAdminSearchUsers } from "@/hooks/use-admin";

export function UserManagement() {
  const intl = useIntl();
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  const { data, isPending } = useAdminSearchUsers({
    q: searchQuery || undefined,
    status: statusFilter || undefined,
  });

  const users = data?.data ?? [];

  return (
    <div className="max-w-content-wide mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "admin.users.title" })}
      />

      {/* Filters */}
      <div className="flex gap-3 mb-6">
        <div className="relative flex-1">
          <Icon
            icon={Search}
            size="sm"
            className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant"
          />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={intl.formatMessage({
              id: "admin.users.searchPlaceholder",
            })}
            className="pl-10"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="bg-surface-container-highest rounded-radius-md px-3 py-2 text-on-surface type-body-sm"
        >
          <option value="">All statuses</option>
          <option value="active">Active</option>
          <option value="suspended">Suspended</option>
          <option value="banned">Banned</option>
        </select>
      </div>

      {/* Loading */}
      {isPending && (
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((n) => (
            <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
          ))}
        </div>
      )}

      {/* Empty */}
      {!isPending && users.length === 0 && (
        <EmptyState
          illustration={<Icon icon={Users} size="xl" />}
          message={intl.formatMessage({ id: "admin.users.empty" })}
          description={intl.formatMessage({
            id: "admin.users.emptyDescription",
          })}
        />
      )}

      {/* User list */}
      <div className="space-y-2">
        {users.map((user) => (
          <RouterLink
            key={user.family_id}
            to={`/admin/users/${user.family_id}`}
            className="block"
          >
            <Card className="p-card-padding hover:bg-surface-container-low transition-colors">
              <div className="flex items-center gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="type-title-sm text-on-surface">
                      {user.family_name}
                    </p>
                    <Badge
                      variant={
                        user.account_status === "active"
                          ? "primary"
                          : user.account_status === "suspended"
                            ? "secondary"
                            : "default"
                      }
                    >
                      {user.account_status}
                    </Badge>
                    <Badge variant="secondary">
                      {user.subscription_tier}
                    </Badge>
                  </div>
                  <p className="type-body-sm text-on-surface-variant mt-0.5">
                    {user.primary_parent_email}
                  </p>
                  <div className="flex items-center gap-4 mt-1 type-label-sm text-on-surface-variant">
                    <span>{user.parent_count} parents</span>
                    <span>{user.student_count} students</span>
                    <span>
                      Joined {new Date(user.created_at).toLocaleDateString()}
                    </span>
                    {user.last_active_at && (
                      <span>
                        Active{" "}
                        {new Date(user.last_active_at).toLocaleDateString()}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </Card>
          </RouterLink>
        ))}
      </div>
    </div>
  );
}
