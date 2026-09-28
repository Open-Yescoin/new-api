-- 03-add-three-token-routes.sql 的短窗口回退。仅在确认无并发修改这五个新渠道、
-- Opus 5.5 型录和五个价格选项后执行；原有 9 款模型及渠道保持不动。
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
DO $$
BEGIN
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_three_token_channels_20260928) <> 5 THEN
    RAISE EXCEPTION 'five-channel record is incomplete';
  END IF;
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_three_token_options_backup_20260928) <> 5 THEN
    RAISE EXCEPTION 'pricing backup is incomplete';
  END IF;
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_three_token_model_20260928) <> 1 THEN
    RAISE EXCEPTION 'new Opus model record is incomplete';
  END IF;
END $$;

DELETE FROM abilities WHERE channel_id IN
  (SELECT id FROM token123_change_log.token123_junxing_three_token_channels_20260928);
DELETE FROM channels WHERE id IN
  (SELECT id FROM token123_change_log.token123_junxing_three_token_channels_20260928);
DELETE FROM models WHERE id IN
  (SELECT id FROM token123_change_log.token123_junxing_three_token_model_20260928);
UPDATE options o SET value=b.value
FROM token123_change_log.token123_junxing_three_token_options_backup_20260928 b
WHERE o.key=b.key;
COMMIT;
