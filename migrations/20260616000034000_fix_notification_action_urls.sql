-- +goose Up
-- Fix dead notification action_urls — routes that never existed in the SPA router.
-- Source handlers updated in internal/notify/service.go.

UPDATE notify_notifications SET action_url = '/friends'
  WHERE action_url = '/friends/requests';

UPDATE notify_notifications SET action_url = '/'
  WHERE action_url = '/dashboard' AND notification_type = 'onboarding_completed';

UPDATE notify_notifications SET action_url = '/learning'
  WHERE action_url = '/dashboard' AND notification_type IN ('activity_streak', 'milestone_achieved');

UPDATE notify_notifications SET action_url = '/learning/reading-lists'
  WHERE action_url = '/learning/reading';

UPDATE notify_notifications SET action_url = '/creator'
  WHERE action_url = '/marketplace/creator/dashboard';

UPDATE notify_notifications SET action_url = '/creator/payouts'
  WHERE action_url = '/marketplace/creator/payouts';

-- +goose Down
UPDATE notify_notifications SET action_url = '/friends/requests'
  WHERE action_url = '/friends' AND notification_type = 'friend_request_sent';

UPDATE notify_notifications SET action_url = '/dashboard'
  WHERE action_url = '/' AND notification_type = 'onboarding_completed';

UPDATE notify_notifications SET action_url = '/dashboard'
  WHERE action_url = '/learning' AND notification_type IN ('activity_streak', 'milestone_achieved');

UPDATE notify_notifications SET action_url = '/learning/reading'
  WHERE action_url = '/learning/reading-lists' AND notification_type = 'book_completed';

UPDATE notify_notifications SET action_url = '/marketplace/creator/dashboard'
  WHERE action_url = '/creator' AND notification_type = 'creator_onboarded';

UPDATE notify_notifications SET action_url = '/marketplace/creator/payouts'
  WHERE action_url = '/creator/payouts' AND notification_type = 'payout_completed';
