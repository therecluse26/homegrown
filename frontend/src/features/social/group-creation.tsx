import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Globe, Lock, UserPlus } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
} from "@/components/ui";
import { FormField } from "@/components/ui/form-field";
import { useToast } from "@/components/ui/toast";
import { useCreateGroup } from "@/hooks/use-social";
import { useState, useEffect, useRef } from "react";

// ─── Join policy options ────────────────────────────────────────────────────

const JOIN_POLICIES = [
  {
    value: "open",
    labelId: "groups.create.joinPolicy.open",
    icon: Globe,
  },
  {
    value: "request",
    labelId: "groups.create.joinPolicy.request",
    icon: UserPlus,
  },
  {
    value: "invite_only",
    labelId: "groups.create.joinPolicy.invite",
    icon: Lock,
  },
] as const;

// ─── Component ─────────────────────────────────────────────────────────────

export function GroupCreation() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { toast } = useToast();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const createGroup = useCreateGroup();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [joinPolicy, setJoinPolicy] = useState("open");

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "groups.create.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  const handleSubmit = () => {
    if (!name.trim()) return;
    createGroup.mutate(
      {
        name: name.trim(),
        description: description.trim() || undefined,
        join_policy: joinPolicy,
      },
      {
        onSuccess: (data) => {
          void navigate(`/groups/${data.summary.id}`);
        },
        onError: () => {
          toast(intl.formatMessage({ id: "groups.create.error" }), "error");
        },
      },
    );
  };

  return (
    <div className="mx-auto max-w-2xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="groups.create.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="groups.create.description" />
      </p>

      <Card>
        <div className="flex flex-col gap-4">
          <FormField
            label={intl.formatMessage({ id: "groups.create.name" })}
          >
            {({ id }) => (
              <Input
                id={id}
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "groups.create.groupDescription",
            })}
          >
            {({ id }) => (
              <Input
                id={id}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            )}
          </FormField>

          <div>
            <p className="type-label-md text-on-surface font-medium mb-2">
              <FormattedMessage id="groups.create.joinPolicy" />
            </p>
            <div className="flex flex-col gap-2" role="radiogroup">
              {JOIN_POLICIES.map((policy) => (
                <label
                  key={policy.value}
                  className={`flex items-center gap-3 p-3 rounded-radius-md cursor-pointer transition-colors ${
                    joinPolicy === policy.value
                      ? "bg-primary/10"
                      : "bg-surface-container-low hover:bg-surface-container-high"
                  }`}
                >
                  <input
                    type="radio"
                    name="joinPolicy"
                    value={policy.value}
                    checked={joinPolicy === policy.value}
                    onChange={() => setJoinPolicy(policy.value)}
                    className="sr-only"
                  />
                  <Icon
                    icon={policy.icon}
                    size="sm"
                    className={
                      joinPolicy === policy.value
                        ? "text-primary"
                        : "text-on-surface-variant"
                    }
                    aria-hidden
                  />
                  <span
                    className={`type-body-sm ${
                      joinPolicy === policy.value
                        ? "text-primary font-medium"
                        : "text-on-surface"
                    }`}
                  >
                    <FormattedMessage id={policy.labelId} />
                  </span>
                  {joinPolicy === policy.value && (
                    <span className="ml-auto w-2 h-2 rounded-radius-full bg-primary" />
                  )}
                </label>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-2 mt-2">
            <Button
              variant="tertiary"
              onClick={() => void navigate("/groups")}
            >
              <FormattedMessage id="action.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleSubmit}
              disabled={!name.trim() || createGroup.isPending}
            >
              <FormattedMessage id="groups.create.submit" />
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}
