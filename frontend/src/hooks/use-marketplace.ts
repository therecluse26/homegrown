import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types (matching Go backend structs — no swagger annotations exist) ──────

// Browse & Listings
export interface ListingBrowseResponse {
  id: string;
  title: string;
  description_preview: string;
  price_cents: number;
  content_type: string;
  thumbnail_url?: string;
  rating_avg: number;
  rating_count: number;
  publisher_name: string;
  creator_store_name: string;
}

export interface ListingFileResponse {
  id: string;
  file_name: string;
  file_size_bytes: number;
  mime_type: string;
  version: number;
}

export interface BundleItemResponse {
  listing_id: string;
  title: string;
  content_type: string;
  price_cents: number;
}

export interface ListingDetailResponse {
  id: string;
  creator_id: string;
  publisher_id: string;
  publisher_name: string;
  title: string;
  description: string;
  price_cents: number;
  methodology_tags: string[];
  subject_tags: string[];
  grade_min?: number;
  grade_max?: number;
  content_type: string;
  worldview_tags: string[];
  preview_url?: string;
  thumbnail_url?: string;
  status: string;
  rating_avg: number;
  rating_count: number;
  version: number;
  files: ListingFileResponse[];
  published_at?: string;
  created_at: string;
  updated_at: string;
  bundle_id?: string;
  bundle_name?: string;
  is_bundle: boolean;
  bundle_items?: BundleItemResponse[];
}

export interface CuratedSectionResponse {
  slug: string;
  display_name: string;
  description?: string;
  listings: ListingBrowseResponse[];
}

export interface AutocompleteResult {
  listing_id: string;
  title: string;
  similarity: number;
}

// Cart
export interface CartItemResponse {
  listing_id: string;
  title: string;
  price_cents: number;
  thumbnail_url?: string;
  added_at: string;
}

export interface CartResponse {
  items: CartItemResponse[];
  total_cents: number;
  item_count: number;
}

export interface CheckoutSessionResponse {
  checkout_url: string;
  payment_session_id: string;
}

// Purchases
export interface PurchaseResponse {
  id: string;
  listing_id: string;
  listing_title: string;
  amount_cents: number;
  refunded: boolean;
  created_at: string;
}

export interface DownloadResponse {
  download_url: string;
  expires_at: string;
}

// Reviews
export interface ReviewResponse {
  id: string;
  listing_id: string;
  rating: number;
  review_text?: string;
  is_anonymous: boolean;
  reviewer_name?: string;
  creator_response?: string;
  creator_response_at?: string;
  created_at: string;
}

// Creator
export interface CreatorResponse {
  id: string;
  parent_id: string;
  onboarding_status: string;
  store_name: string;
  store_bio?: string;
  store_logo_url?: string;
  store_banner_url?: string;
  created_at: string;
}

export interface SaleSummary {
  purchase_id: string;
  listing_title: string;
  amount_cents: number;
  creator_payout_cents: number;
  purchased_at: string;
}

export interface CreatorDashboardResponse {
  total_sales_count: number;
  total_earnings_cents: number;
  period_sales_count: number;
  period_earnings_cents: number;
  pending_payout_cents: number;
  average_rating: number;
  total_reviews: number;
  recent_sales: SaleSummary[];
}

export interface PublisherResponse {
  id: string;
  name: string;
  slug: string;
  description?: string;
  logo_url?: string;
  website_url?: string;
  is_verified: boolean;
  member_count: number;
}

// Commands
export interface CreateListingCommand {
  publisher_id: string;
  title: string;
  description: string;
  price_cents: number;
  methodology_tags: string[];
  subject_tags: string[];
  grade_min?: number;
  grade_max?: number;
  content_type: string;
  worldview_tags?: string[];
  preview_url?: string;
  thumbnail_url?: string;
}

export interface UpdateListingCommand {
  title?: string;
  description?: string;
  price_cents?: number;
  methodology_tags?: string[];
  subject_tags?: string[];
  grade_min?: number;
  grade_max?: number;
  worldview_tags?: string[];
  preview_url?: string;
  thumbnail_url?: string;
  change_summary?: string;
}

export interface CreateReviewCommand {
  rating: number;
  review_text?: string;
  is_anonymous?: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

export interface BrowseParams {
  q?: string;
  methodology_ids?: string[];
  subject_slugs?: string[];
  grade_min?: number;
  grade_max?: number;
  content_type?: string;
  worldview_tags?: string[];
  price_min?: number;
  price_max?: number;
  min_rating?: number;
  sort_by?: string;
  cursor?: string;
  limit?: number;
}

// ─── Browse & Discovery ─────────────────────────────────────────────────────

function buildBrowseQuery(params: BrowseParams): string {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.content_type) qs.set("content_type", params.content_type);
  if (params.sort_by) qs.set("sort_by", params.sort_by);
  if (params.cursor) qs.set("cursor", params.cursor);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.grade_min != null) qs.set("grade_min", String(params.grade_min));
  if (params.grade_max != null) qs.set("grade_max", String(params.grade_max));
  if (params.price_min != null) qs.set("price_min", String(params.price_min));
  if (params.price_max != null) qs.set("price_max", String(params.price_max));
  if (params.min_rating != null) qs.set("min_rating", String(params.min_rating));
  params.methodology_ids?.forEach((id) => qs.append("methodology_ids", id));
  params.subject_slugs?.forEach((s) => qs.append("subject_slugs", s));
  params.worldview_tags?.forEach((w) => qs.append("worldview_tags", w));
  const str = qs.toString();
  return str ? `?${str}` : "";
}

export function useBrowseListings(params: BrowseParams = {}) {
  return useQuery({
    queryKey: ["marketplace", "browse", params],
    queryFn: () =>
      apiClient<PaginatedResponse<ListingBrowseResponse>>(
        `/v1/marketplace/listings${buildBrowseQuery(params)}`,
      ),
  });
}

export function useListingDetail(listingId: string | undefined) {
  return useQuery({
    queryKey: ["marketplace", "listing", listingId],
    queryFn: () =>
      apiClient<ListingDetailResponse>(
        `/v1/marketplace/listings/${listingId}`,
      ),
    enabled: !!listingId,
  });
}

export function useListingReviews(listingId: string | undefined) {
  return useQuery({
    queryKey: ["marketplace", "reviews", listingId],
    queryFn: () =>
      apiClient<PaginatedResponse<ReviewResponse>>(
        `/v1/marketplace/listings/${listingId}/reviews`,
      ),
    enabled: !!listingId,
  });
}

export function useCuratedSections() {
  return useQuery({
    queryKey: ["marketplace", "curated"],
    queryFn: () =>
      apiClient<CuratedSectionResponse[]>("/v1/marketplace/curated-sections"),
  });
}

export function useAutocompleteListings(q: string) {
  return useQuery({
    queryKey: ["marketplace", "autocomplete", q],
    queryFn: () =>
      apiClient<AutocompleteResult[]>(
        `/v1/marketplace/listings/autocomplete?q=${encodeURIComponent(q)}`,
      ),
    enabled: q.length >= 2,
  });
}

// ─── Cart ────────────────────────────────────────────────────────────────────

export function useCart() {
  return useQuery({
    queryKey: ["marketplace", "cart"],
    queryFn: () => apiClient<CartResponse>("/v1/marketplace/cart"),
  });
}

export function useAddToCart() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient<void>("/v1/marketplace/cart/items", {
        method: "POST",
        body: { listing_id: listingId },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "cart"] });
    },
  });
}

export function useRemoveFromCart() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient<void>(`/v1/marketplace/cart/items/${listingId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "cart"] });
    },
  });
}

export function useCheckout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<CheckoutSessionResponse>("/v1/marketplace/cart/checkout", {
        method: "POST",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "cart"] });
    },
  });
}

// ─── Purchases ───────────────────────────────────────────────────────────────

export function usePurchases() {
  return useQuery({
    queryKey: ["marketplace", "purchases"],
    queryFn: () =>
      apiClient<PaginatedResponse<PurchaseResponse>>("/v1/marketplace/purchases"),
  });
}

export function useDownloadURL(listingId: string, fileId: string) {
  return useQuery({
    queryKey: ["marketplace", "download", listingId, fileId],
    queryFn: () =>
      apiClient<DownloadResponse>(
        `/v1/marketplace/purchases/${listingId}/download/${fileId}`,
      ),
    enabled: false, // manual trigger
  });
}

export function useGetFreeListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient<void>(`/v1/marketplace/listings/${listingId}/get`, {
        method: "POST",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "purchases"] });
    },
  });
}

export function usePurchaseBundle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (bundleId: string) =>
      apiClient<{ purchase_id: string }>(
        `/v1/marketplace/bundles/${bundleId}/purchase`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["marketplace", "purchases"] });
      void qc.invalidateQueries({ queryKey: ["marketplace", "cart"] });
    },
  });
}

// ─── Reviews ─────────────────────────────────────────────────────────────────

export function useCreateReview(listingId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: CreateReviewCommand) =>
      apiClient<ReviewResponse>(
        `/v1/marketplace/listings/${listingId}/reviews`,
        { method: "POST", body: cmd },
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "reviews", listingId] });
    },
  });
}

export function useDeleteReview() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (reviewId: string) =>
      apiClient<void>(`/v1/marketplace/reviews/${reviewId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "reviews"] });
    },
  });
}

// ─── Creator ─────────────────────────────────────────────────────────────────

export function useCreatorProfile() {
  return useQuery({
    queryKey: ["marketplace", "creator", "me"],
    queryFn: () =>
      apiClient<CreatorResponse>("/v1/marketplace/creators/me"),
  });
}

export function useCreatorDashboard(period = "last_30_days") {
  return useQuery({
    queryKey: ["marketplace", "creator", "dashboard", period],
    queryFn: () =>
      apiClient<CreatorDashboardResponse>(
        `/v1/marketplace/creators/dashboard?period=${period}`,
      ),
  });
}

export function useCreatorListings(status?: string) {
  const qs = status ? `?status=${status}` : "";
  return useQuery({
    queryKey: ["marketplace", "creator", "listings", status],
    queryFn: () =>
      apiClient<PaginatedResponse<ListingDetailResponse>>(
        `/v1/marketplace/creators/listings${qs}`,
      ),
  });
}

export function useCreateListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: CreateListingCommand) =>
      apiClient<ListingDetailResponse>("/v1/marketplace/listings", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "creator", "listings"] });
    },
  });
}

export function useUpdateListing(listingId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: UpdateListingCommand) =>
      apiClient<ListingDetailResponse>(
        `/v1/marketplace/listings/${listingId}`,
        { method: "PUT", body: cmd },
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "listing", listingId] });
      qc.invalidateQueries({ queryKey: ["marketplace", "creator", "listings"] });
    },
  });
}

export function useSubmitListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient<void>(`/v1/marketplace/listings/${listingId}/submit`, {
        method: "POST",
      }),
    onSuccess: (_d, listingId) => {
      qc.invalidateQueries({ queryKey: ["marketplace", "listing", listingId] });
      qc.invalidateQueries({ queryKey: ["marketplace", "creator", "listings"] });
    },
  });
}

export function usePublishListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient<void>(`/v1/marketplace/listings/${listingId}/publish`, {
        method: "POST",
      }),
    onSuccess: (_d, listingId) => {
      qc.invalidateQueries({ queryKey: ["marketplace", "listing", listingId] });
      qc.invalidateQueries({ queryKey: ["marketplace", "creator", "listings"] });
    },
  });
}

export function useArchiveListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient<void>(`/v1/marketplace/listings/${listingId}/archive`, {
        method: "POST",
      }),
    onSuccess: (_d, listingId) => {
      qc.invalidateQueries({ queryKey: ["marketplace", "listing", listingId] });
      qc.invalidateQueries({ queryKey: ["marketplace", "creator", "listings"] });
    },
  });
}

export function useRegisterCreator() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      store_name: string;
      store_bio?: string;
      tos_accepted: boolean;
    }) =>
      apiClient<CreatorResponse>("/v1/marketplace/creators/register", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "creator"] });
    },
  });
}

export function useRequestPayout() {
  return useMutation({
    mutationFn: () =>
      apiClient<{ payout_id: string; amount_cents: number; status: string }>(
        "/v1/marketplace/payouts/request",
        { method: "POST" },
      ),
  });
}

// ─── Payout Methods & Config ──────────────────────────────────────────────────

export interface PayoutMethod {
  id: string;
  type: "bank_account" | "paypal";
  label: string;
  is_default: boolean;
  last_four: string;
}

export interface PayoutHistory {
  id: string;
  amount_cents: number;
  status: "pending" | "processing" | "completed" | "failed";
  created_at: string;
}

export interface PayoutConfig {
  minimum_threshold_cents: number;
  next_payout_date: string;
  total_earnings_cents: number;
  pending_balance_cents: number;
}

export function usePayoutConfig() {
  return useQuery({
    queryKey: ["marketplace", "payouts", "config"],
    queryFn: () =>
      apiClient<PayoutConfig>("/v1/marketplace/creators/payouts/config"),
  });
}

export function usePayoutMethods() {
  return useQuery({
    queryKey: ["marketplace", "payouts", "methods"],
    queryFn: () =>
      apiClient<PayoutMethod[]>("/v1/marketplace/creators/payouts/methods"),
  });
}

export function usePayoutHistory() {
  return useQuery({
    queryKey: ["marketplace", "payouts", "history"],
    queryFn: () =>
      apiClient<PayoutHistory[]>("/v1/marketplace/creators/payouts/history"),
  });
}

export function useAddPayoutMethod() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      type: "bank_account" | "paypal";
      label: string;
      account_number?: string;
      routing_number?: string;
      paypal_email?: string;
    }) =>
      apiClient<PayoutMethod>("/v1/marketplace/creators/payouts/methods", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "payouts", "methods"] });
    },
  });
}

export function useRemovePayoutMethod() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (methodId: string) =>
      apiClient<void>(`/v1/marketplace/creators/payouts/methods/${methodId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "payouts", "methods"] });
    },
  });
}

export function useSetDefaultPayoutMethod() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (methodId: string) =>
      apiClient<void>(`/v1/marketplace/creators/payouts/methods/${methodId}/default`, {
        method: "PUT",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["marketplace", "payouts", "methods"] });
    },
  });
}

// ─── Creator Verification ─────────────────────────────────────────────────────

export interface CreatorVerification {
  status: "unverified" | "pending" | "verified" | "rejected";
  legal_name?: string;
  tax_id_last_four?: string;
  submitted_at?: string;
}

export function useCreatorVerification() {
  return useQuery({
    queryKey: ["marketplace", "creator", "verification"],
    queryFn: () =>
      apiClient<CreatorVerification>("/v1/marketplace/creators/verification"),
  });
}

export function useSubmitVerification() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: { legal_name: string; tax_id: string }) =>
      apiClient<CreatorVerification>("/v1/marketplace/creators/verification", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ["marketplace", "creator", "verification"],
      });
    },
  });
}

// ─── Listing Version History ──────────────────────────────────────────────────

export interface ListingVersion {
  id: string;
  version_number: number;
  file_name: string;
  file_size_bytes: number;
  uploaded_at: string;
  is_current: boolean;
}

export function useListingVersions(listingId: string) {
  return useQuery({
    queryKey: ["marketplace", "listing", listingId, "versions"],
    queryFn: () =>
      apiClient<ListingVersion[]>(
        `/v1/marketplace/listings/${listingId}/versions`,
      ),
    enabled: !!listingId,
  });
}

// ─── Creator Reviews ──────────────────────────────────────────────────────────

export interface CreatorReview {
  id: string;
  listing_id: string;
  listing_title: string;
  reviewer_name: string;
  rating: number;
  text: string;
  response?: string;
  created_at: string;
}

export function useCreatorReviews() {
  return useQuery({
    queryKey: ["marketplace", "creator", "reviews"],
    queryFn: () =>
      apiClient<CreatorReview[]>("/v1/marketplace/creators/reviews"),
  });
}

export function useRespondToReview() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ reviewId, response }: { reviewId: string; response: string }) =>
      apiClient<void>(`/v1/marketplace/reviews/${reviewId}/respond`, {
        method: "POST",
        body: { response },
      }),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ["marketplace", "creator", "reviews"],
      });
    },
  });
}
