-- 04-route-gpt6-sol-legacy-host.sql 的短窗口回退。执行前确认相关渠道、能力
-- 与 DynamicTextPricing 没有新的并发变更；回退后 gpt-6-sol 再次走 api.centos.hk。
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
DO $$
BEGIN
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_gpt6_legacy_channels_backup_20260928) <> 3
     OR (SELECT count(*) FROM token123_change_log.token123_junxing_gpt6_legacy_abilities_backup_20260928) <> 3
     OR (SELECT count(*) FROM token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928) <> 3 THEN
    RAISE EXCEPTION 'GPT 6 Sol backup is incomplete';
  END IF;
END $$;

DELETE FROM abilities WHERE channel_id IN
  (SELECT id FROM token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928);
DELETE FROM channels WHERE id IN
  (SELECT id FROM token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928);
UPDATE channels c SET models=b.models FROM
  token123_change_log.token123_junxing_gpt6_legacy_channels_backup_20260928 b WHERE c.id=b.id;
INSERT INTO abilities SELECT * FROM token123_change_log.token123_junxing_gpt6_legacy_abilities_backup_20260928;
UPDATE options o SET value=b.value FROM
  token123_change_log.token123_junxing_gpt6_legacy_dynamic_backup_20260928 b WHERE o.key=b.key;
COMMIT;
