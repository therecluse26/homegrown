import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────
// Social types are not yet in the generated schema, so we define lightweight
// frontend-only types matching the Go backend structs. These will be replaced
// with generated types once backend swagger annotations are added.

// ── Post Types ──────────────────────────────────────────────────────────────

export type PostType =
  | "text"
  | "photo"
  | "milestone"
  | "event_share"
  | "marketplace_review"
  | "resource_share";

export interface PostResponse {
  id: string;
  family_id: string;
  author_name: string;
  author_photo_url?: string;
  post_type: PostType;
  content?: string;
  attachments?: unknown;
  group_id?: string;
  group_name?: string;
  visibility: string;
  likes_count: number;
  comments_count: number;
  is_edited: boolean;
  is_liked_by_me: boolean;
  created_at: string;
}

export interface PostDetailResponse {
  post: PostResponse;
  comments: CommentResponse[];
}

export interface FeedResponse {
  posts: PostResponse[];
  next_cursor?: string;
}

export interface CreatePostCommand {
  post_type: PostType;
  content?: string;
  attachments?: unknown;
  group_id?: string;
}

export interface UpdatePostCommand {
  content?: string;
  attachments?: unknown;
}

// ── Comment Types ───────────────────────────────────────────────────────────

export interface CommentResponse {
  id: string;
  post_id: string;
  family_id: string;
  author_name: string;
  author_photo_url?: string;
  parent_comment_id?: string;
  content: string;
  created_at: string;
  replies?: CommentResponse[];
}

export interface CreateCommentCommand {
  content: string;
  parent_comment_id?: string;
}

// ── Friend Types ────────────────────────────────────────────────────────────

export interface FriendResponse {
  family_id: string;
  display_name: string;
  profile_photo_url?: string;
  methodology_names?: string[];
  friends_since: string;
}

export interface FriendshipResponse {
  id: string;
  requester_family_id: string;
  accepter_family_id: string;
  status: string;
  created_at: string;
}

export interface FriendRequestResponse {
  friendship_id: string;
  family_id: string;
  display_name: string;
  profile_photo_url?: string;
  created_at: string;
}

export interface BlockedFamilyResponse {
  family_id: string;
  display_name: string;
  blocked_at: string;
}

// ── Group Types ─────────────────────────────────────────────────────────────

export interface GroupSummaryResponse {
  id: string;
  group_type: string;
  name: string;
  description?: string;
  cover_photo_url?: string;
  methodology_name?: string;
  join_policy: string;
  member_count: number;
  is_member: boolean;
}

export interface GroupDetailResponse {
  summary: GroupSummaryResponse;
  creator_family_id?: string;
  my_role?: string;
  my_status?: string;
  created_at: string;
}

export interface GroupMemberResponse {
  family_id: string;
  display_name: string;
  role: string;
  status: string;
  joined_at?: string;
}

export interface CreateGroupCommand {
  name: string;
  description?: string;
  cover_photo_url?: string;
  join_policy: string;
  methodology_slug?: string;
}

export interface UpdateGroupCommand {
  name?: string;
  description?: string;
  cover_photo_url?: string;
  join_policy?: string;
}

// ── Event Types ─────────────────────────────────────────────────────────────

export interface EventSummaryResponse {
  id: string;
  title: string;
  event_date: string;
  end_date?: string;
  location_name?: string;
  location_region?: string;
  is_virtual: boolean;
  creator_family_name: string;
  capacity?: number;
  visibility: string;
  status: string;
  attendee_count: number;
  my_rsvp?: string;
}

export interface EventRsvpResponse {
  family_id: string;
  display_name: string;
  status: string;
  created_at: string;
}

export interface EventDetailResponse extends EventSummaryResponse {
  creator_family_id: string;
  group_id?: string;
  group_name?: string;
  description?: string;
  virtual_url?: string;
  methodology_name?: string;
  rsvps?: EventRsvpResponse[];
  created_at: string;
}

export interface CreateEventCommand {
  title: string;
  description?: string;
  event_date: string;
  end_date?: string;
  location_name?: string;
  location_region?: string;
  is_virtual: boolean;
  virtual_url?: string;
  capacity?: number;
  visibility: string;
  group_id?: string;
  methodology_slug?: string;
}

export interface UpdateEventCommand {
  title?: string;
  description?: string;
  event_date?: string;
  end_date?: string;
  location_name?: string;
  location_region?: string;
  is_virtual?: boolean;
  virtual_url?: string;
  capacity?: number;
  visibility?: string;
}

export interface RSVPCommand {
  status: "going" | "interested" | "not_going";
}

// ── Conversation / Message Types ────────────────────────────────────────────

export interface ConversationSummaryResponse {
  id: string;
  other_parent_name: string;
  last_message_preview?: string;
  unread_count: number;
  is_muted: boolean;
  updated_at: string;
}

export interface ConversationParticipantResponse {
  parent_id: string;
  family_id: string;
  display_name: string;
  profile_photo_url?: string;
}

export interface ConversationResponse {
  id: string;
  participants: ConversationParticipantResponse[];
  updated_at: string;
}

export interface MessageResponse {
  id: string;
  conversation_id: string;
  sender_parent_id: string;
  sender_name: string;
  content: string;
  attachments?: unknown;
  created_at: string;
}

export interface SendMessageCommand {
  content: string;
  attachments?: unknown;
}

// ── Profile Types ───────────────────────────────────────────────────────────

export interface ProfileChildResponse {
  display_name: string;
  grade_level?: string;
}

export interface ProfileResponse {
  family_id: string;
  display_name?: string;
  bio?: string;
  profile_photo_url?: string;
  parent_names?: string[];
  children?: ProfileChildResponse[];
  methodology_names?: string[];
  location_region?: string;
  location_visible?: boolean;
  privacy_settings?: PrivacySettings;
  friendship_status?: string;
  is_friend: boolean;
}

export interface PrivacySettings {
  display_name: string;
  parent_names: string;
  children_names: string;
  children_ages: string;
  location: string;
  methodology: string;
}

export interface UpdateProfileCommand {
  bio?: string;
  profile_photo_url?: string;
  privacy_settings?: PrivacySettings;
  location_visible?: boolean;
}

// ── Discovery Types ─────────────────────────────────────────────────────────

export interface DiscoverableFamilyResponse {
  family_id: string;
  display_name: string;
  profile_photo_url?: string;
  methodology_names?: string[];
  location_region?: string;
}

// ─── Feed & Post Queries ────────────────────────────────────────────────────

export function useFeed(params?: { offset?: number; limit?: number }) {
  const offset = params?.offset ?? 0;
  const limit = params?.limit ?? 20;
  return useQuery({
    queryKey: ["social", "feed", { offset, limit }],
    queryFn: () =>
      apiClient<FeedResponse>(
        `/v1/social/feed?offset=${offset}&limit=${limit}`,
      ),
    staleTime: 1000 * 30,
  });
}

export function usePostDetail(postId: string | undefined) {
  return useQuery({
    queryKey: ["social", "posts", postId],
    queryFn: () =>
      apiClient<PostDetailResponse>(`/v1/social/posts/${postId ?? ""}`),
    enabled: !!postId,
  });
}

export function useCreatePost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreatePostCommand) =>
      apiClient<PostResponse>("/v1/social/posts", {
        method: "POST",
        body: data,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "feed"] });
    },
  });
}

export function useUpdatePost(postId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdatePostCommand) =>
      apiClient<PostResponse>(`/v1/social/posts/${postId}`, {
        method: "PATCH",
        body: data,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "posts", postId] });
      void qc.invalidateQueries({ queryKey: ["social", "feed"] });
    },
  });
}

export function useDeletePost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) =>
      apiClient<void>(`/v1/social/posts/${postId}`, { method: "DELETE" }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "feed"] });
    },
  });
}

export function useLikePost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) =>
      apiClient<void>(`/v1/social/posts/${postId}/like`, { method: "POST" }),
    onSuccess: (_data, postId) => {
      void qc.invalidateQueries({ queryKey: ["social", "posts", postId] });
      void qc.invalidateQueries({ queryKey: ["social", "feed"] });
    },
  });
}

export function useUnlikePost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) =>
      apiClient<void>(`/v1/social/posts/${postId}/like`, { method: "DELETE" }),
    onSuccess: (_data, postId) => {
      void qc.invalidateQueries({ queryKey: ["social", "posts", postId] });
      void qc.invalidateQueries({ queryKey: ["social", "feed"] });
    },
  });
}

// ─── Comment Queries ────────────────────────────────────────────────────────

export function useComments(postId: string | undefined) {
  return useQuery({
    queryKey: ["social", "posts", postId, "comments"],
    queryFn: () =>
      apiClient<CommentResponse[]>(
        `/v1/social/posts/${postId ?? ""}/comments`,
      ),
    enabled: !!postId,
  });
}

export function useCreateComment(postId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateCommentCommand) =>
      apiClient<CommentResponse>(`/v1/social/posts/${postId}/comments`, {
        method: "POST",
        body: data,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["social", "posts", postId, "comments"],
      });
      void qc.invalidateQueries({ queryKey: ["social", "posts", postId] });
      void qc.invalidateQueries({ queryKey: ["social", "feed"] });
    },
  });
}

export function useDeleteComment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (commentId: string) =>
      apiClient<void>(`/v1/social/comments/${commentId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social"] });
    },
  });
}

// ─── Friend Queries ─────────────────────────────────────────────────────────

export function useFriends(params?: { cursor?: string; limit?: number }) {
  return useQuery({
    queryKey: ["social", "friends", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      if (params?.cursor) searchParams.set("cursor", params.cursor);
      if (params?.limit) searchParams.set("limit", String(params.limit));
      const qs = searchParams.toString();
      return apiClient<FriendResponse[]>(
        `/v1/social/friends${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60,
  });
}

export function useIncomingFriendRequests() {
  return useQuery({
    queryKey: ["social", "friends", "requests", "incoming"],
    queryFn: () =>
      apiClient<FriendRequestResponse[]>(
        "/v1/social/friends/requests/incoming",
      ),
    staleTime: 1000 * 30,
  });
}

export function useOutgoingFriendRequests() {
  return useQuery({
    queryKey: ["social", "friends", "requests", "outgoing"],
    queryFn: () =>
      apiClient<FriendRequestResponse[]>(
        "/v1/social/friends/requests/outgoing",
      ),
  });
}

export function useSendFriendRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (familyId: string) =>
      apiClient<FriendshipResponse>(
        `/v1/social/friends/request/${familyId}`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["social", "friends", "requests"],
      });
    },
  });
}

export function useAcceptFriendRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (friendshipId: string) =>
      apiClient<FriendshipResponse>(
        `/v1/social/friends/accept/${friendshipId}`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "friends"] });
    },
  });
}

export function useRejectFriendRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (friendshipId: string) =>
      apiClient<void>(`/v1/social/friends/reject/${friendshipId}`, {
        method: "POST",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "friends"] });
    },
  });
}

export function useUnfriend() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (familyId: string) =>
      apiClient<void>(`/v1/social/friends/${familyId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "friends"] });
      void qc.invalidateQueries({ queryKey: ["social", "feed"] });
    },
  });
}

// ─── Block Queries ──────────────────────────────────────────────────────────

export function useBlockedFamilies() {
  return useQuery({
    queryKey: ["social", "blocks"],
    queryFn: () =>
      apiClient<BlockedFamilyResponse[]>("/v1/social/blocks"),
  });
}

export function useBlockFamily() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (familyId: string) =>
      apiClient<void>(`/v1/social/blocks/${familyId}`, { method: "POST" }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social"] });
    },
  });
}

export function useUnblockFamily() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (familyId: string) =>
      apiClient<void>(`/v1/social/blocks/${familyId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "blocks"] });
    },
  });
}

// ─── Group Queries ──────────────────────────────────────────────────────────

export function useMyGroups() {
  return useQuery({
    queryKey: ["social", "groups", "mine"],
    queryFn: () => apiClient<GroupDetailResponse[]>("/v1/social/groups"),
    staleTime: 1000 * 60,
  });
}

export function usePlatformGroups() {
  return useQuery({
    queryKey: ["social", "groups", "platform"],
    queryFn: () =>
      apiClient<GroupDetailResponse[]>("/v1/social/groups/platform"),
    staleTime: 1000 * 60 * 5,
  });
}

export function useGroupDetail(groupId: string | undefined) {
  return useQuery({
    queryKey: ["social", "groups", groupId],
    queryFn: () =>
      apiClient<GroupDetailResponse>(`/v1/social/groups/${groupId ?? ""}`),
    enabled: !!groupId,
  });
}

export function useGroupMembers(groupId: string | undefined) {
  return useQuery({
    queryKey: ["social", "groups", groupId, "members"],
    queryFn: () =>
      apiClient<GroupMemberResponse[]>(
        `/v1/social/groups/${groupId ?? ""}/members`,
      ),
    enabled: !!groupId,
  });
}

export function useGroupPosts(
  groupId: string | undefined,
  params?: { offset?: number; limit?: number },
) {
  const offset = params?.offset ?? 0;
  const limit = params?.limit ?? 20;
  return useQuery({
    queryKey: ["social", "groups", groupId, "posts", { offset, limit }],
    queryFn: () =>
      apiClient<PostResponse[]>(
        `/v1/social/groups/${groupId ?? ""}/posts?offset=${offset}&limit=${limit}`,
      ),
    enabled: !!groupId,
  });
}

export function useJoinGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (groupId: string) =>
      apiClient<void>(`/v1/social/groups/${groupId}/join`, {
        method: "POST",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "groups"] });
    },
  });
}

export function useLeaveGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (groupId: string) =>
      apiClient<void>(`/v1/social/groups/${groupId}/leave`, {
        method: "POST",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "groups"] });
    },
  });
}

// ─── Event Queries ──────────────────────────────────────────────────────────

export function useEvents(params?: { offset?: number; limit?: number }) {
  const offset = params?.offset ?? 0;
  const limit = params?.limit ?? 20;
  return useQuery({
    queryKey: ["social", "events", { offset, limit }],
    queryFn: () =>
      apiClient<EventDetailResponse[]>(
        `/v1/social/events?offset=${offset}&limit=${limit}`,
      ),
    staleTime: 1000 * 60,
  });
}

export function useEventDetail(eventId: string | undefined) {
  return useQuery({
    queryKey: ["social", "events", eventId],
    queryFn: () =>
      apiClient<EventDetailResponse>(`/v1/social/events/${eventId ?? ""}`),
    enabled: !!eventId,
  });
}

export function useCreateEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateEventCommand) =>
      apiClient<EventDetailResponse>("/v1/social/events", {
        method: "POST",
        body: data,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "events"] });
    },
  });
}

export function useUpdateEvent(eventId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateEventCommand) =>
      apiClient<EventDetailResponse>(`/v1/social/events/${eventId}`, {
        method: "PATCH",
        body: data,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "events", eventId] });
      void qc.invalidateQueries({ queryKey: ["social", "events"] });
    },
  });
}

export function useCancelEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (eventId: string) =>
      apiClient<void>(`/v1/social/events/${eventId}/cancel`, {
        method: "POST",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "events"] });
    },
  });
}

export function useRSVP(eventId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: RSVPCommand) =>
      apiClient<void>(`/v1/social/events/${eventId}/rsvp`, {
        method: "POST",
        body: data,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "events", eventId] });
      void qc.invalidateQueries({ queryKey: ["social", "events"] });
    },
  });
}

export function useRemoveRSVP(eventId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>(`/v1/social/events/${eventId}/rsvp`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "events", eventId] });
    },
  });
}

// ─── Conversation & Message Queries ─────────────────────────────────────────

export function useConversations(params?: {
  offset?: number;
  limit?: number;
}) {
  const offset = params?.offset ?? 0;
  const limit = params?.limit ?? 20;
  return useQuery({
    queryKey: ["social", "conversations", { offset, limit }],
    queryFn: () =>
      apiClient<ConversationSummaryResponse[]>(
        `/v1/social/conversations?offset=${offset}&limit=${limit}`,
      ),
    staleTime: 1000 * 15,
  });
}

export function useMessages(
  conversationId: string | undefined,
  params?: { offset?: number; limit?: number },
) {
  const offset = params?.offset ?? 0;
  const limit = params?.limit ?? 50;
  return useQuery({
    queryKey: ["messages", conversationId, { offset, limit }],
    queryFn: () =>
      apiClient<MessageResponse[]>(
        `/v1/social/conversations/${conversationId ?? ""}/messages?offset=${offset}&limit=${limit}`,
      ),
    enabled: !!conversationId,
    staleTime: 1000 * 10,
  });
}

export function useCreateConversation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (recipientParentId: string) =>
      apiClient<ConversationResponse>("/v1/social/conversations", {
        method: "POST",
        body: { recipient_parent_id: recipientParentId },
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "conversations"] });
    },
  });
}

export function useSendMessage(conversationId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: SendMessageCommand) =>
      apiClient<MessageResponse>(
        `/v1/social/conversations/${conversationId}/messages`,
        { method: "POST", body: data },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["messages", conversationId],
      });
      void qc.invalidateQueries({ queryKey: ["social", "conversations"] });
    },
  });
}

export function useMarkConversationRead(conversationId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>(
        `/v1/social/conversations/${conversationId}/read`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "conversations"] });
    },
  });
}

export function useMuteConversation(conversationId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>(
        `/v1/social/conversations/${conversationId}/mute`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "conversations"] });
    },
  });
}

export function useUnmuteConversation(conversationId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>(
        `/v1/social/conversations/${conversationId}/mute`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social", "conversations"] });
    },
  });
}

export function useUpdateComment(commentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { content: string }) =>
      apiClient<CommentResponse>(`/v1/social/comments/${commentId}`, {
        method: "PATCH",
        body: data,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["social"] });
    },
  });
}

export function useEventRsvps(eventId: string | undefined) {
  return useQuery({
    queryKey: ["social", "events", eventId, "rsvps"],
    queryFn: () =>
      apiClient<EventRsvpResponse[]>(
        `/v1/social/events/${eventId ?? ""}/rsvps`,
      ),
    enabled: !!eventId,
  });
}

// ─── Profile Queries ────────────────────────────────────────────────────────

export function useMyProfile() {
  return useQuery({
    queryKey: ["social", "profile"],
    queryFn: () => apiClient<ProfileResponse>("/v1/social/profile"),
    staleTime: 1000 * 60 * 2,
  });
}

export function useFamilyProfileView(familyId: string | undefined) {
  return useQuery({
    queryKey: ["social", "families", familyId, "profile"],
    queryFn: () =>
      apiClient<ProfileResponse>(
        `/v1/social/families/${familyId ?? ""}/profile`,
      ),
    enabled: !!familyId,
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateProfileCommand) =>
      apiClient<ProfileResponse>("/v1/social/profile", {
        method: "PATCH",
        body,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(["social", "profile"], data);
    },
  });
}

// ─── Discovery Queries ──────────────────────────────────────────────────────

export function useDiscoverFamilies(params?: {
  methodology_slug?: string;
}) {
  return useQuery({
    queryKey: ["social", "discover", "families", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      if (params?.methodology_slug)
        searchParams.set("methodology_slug", params.methodology_slug);
      const qs = searchParams.toString();
      return apiClient<DiscoverableFamilyResponse[]>(
        `/v1/social/discover/families${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60 * 5,
  });
}

export function useDiscoverEvents(params?: {
  methodology_slug?: string;
  location_region?: string;
}) {
  return useQuery({
    queryKey: ["social", "discover", "events", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      if (params?.methodology_slug)
        searchParams.set("methodology_slug", params.methodology_slug);
      if (params?.location_region)
        searchParams.set("location_region", params.location_region);
      const qs = searchParams.toString();
      return apiClient<EventSummaryResponse[]>(
        `/v1/social/discover/events${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60 * 5,
  });
}

export function useDiscoverGroups(params?: {
  methodology_slug?: string;
}) {
  return useQuery({
    queryKey: ["social", "discover", "groups", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      if (params?.methodology_slug)
        searchParams.set("methodology_slug", params.methodology_slug);
      const qs = searchParams.toString();
      return apiClient<GroupSummaryResponse[]>(
        `/v1/social/discover/groups${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60 * 5,
  });
}
