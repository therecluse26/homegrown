import { Avatar } from "../ui/avatar";

type UserAvatarProps = {
  name: string;
  avatarUrl?: string;
  size?: "xs" | "sm" | "md" | "lg" | "xl";
  className?: string;
};

export function UserAvatar({
  name,
  avatarUrl,
  size = "md",
  className = "",
}: UserAvatarProps) {
  return (
    <Avatar name={name} src={avatarUrl} size={size} className={className} />
  );
}
