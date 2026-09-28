-- 02-refresh-nine-models.sql 的短窗口回退。仅在确认没有并发修改相同渠道、
-- OpenAI/Anthropic 型录、能力和这 6 个价格选项后执行；保留备份表供审计。
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
DO $$
BEGIN
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_models_channels_backup_20260928) <> 20 THEN
    RAISE EXCEPTION 'channel backup is incomplete';
  END IF;
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_models_options_backup_20260928) <> 6 THEN
    RAISE EXCEPTION 'option backup is incomplete';
  END IF;
END $$;

UPDATE channels c SET status=b.status,"group"=b."group",models=b.models,base_url=b.base_url
FROM token123_change_log.token123_junxing_models_channels_backup_20260928 b WHERE c.id=b.id;
DELETE FROM abilities a WHERE a.channel_id IN (SELECT id FROM token123_change_log.token123_junxing_models_channels_backup_20260928)
 OR a.model IN (SELECT model_name FROM token123_change_log.token123_junxing_models_catalog_backup_20260928);
INSERT INTO abilities SELECT * FROM token123_change_log.token123_junxing_models_abilities_backup_20260928;
DELETE FROM models m WHERE m.vendor_id IN (1,2)
 AND NOT EXISTS (SELECT 1 FROM token123_change_log.token123_junxing_models_catalog_backup_20260928 b WHERE b.id=m.id)
 AND m.model_name IN ('gpt-6-astra','gpt-6-sol','claude-fable-5-1','gpt-image-2.5-flare','gpt-image-2.5-sunburst');
UPDATE models m SET status=b.status,deleted_at=b.deleted_at,updated_time=b.updated_time
FROM token123_change_log.token123_junxing_models_catalog_backup_20260928 b WHERE m.id=b.id;
UPDATE options o SET value=b.value
FROM token123_change_log.token123_junxing_models_options_backup_20260928 b WHERE o.key=b.key;
COMMIT;
