-- Token123 / 君的星辰：gpt-6-sol 在 api.centos.hk 持续 503，三枚令牌在
-- ai.centos.hk 的真实流式请求均成功。仅此模型改走旧域名的三条独立渠道。
-- 其余 GPT 仍走 api.centos.hk 以规避原有 gpt-5.6-sol 524。
-- 同一 gpt-pro 0.4x 上游组和三枚原有密钥；成本及 2.2x 售价不变。
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

DO $$
BEGIN
  IF (SELECT count(*) FROM channels WHERE id IN (11,35,36) AND status=1
      AND base_url='https://api.centos.hk'
      AND models='gpt-5.6-sol,gpt-6-astra,gpt-5.6-terra,gpt-6-sol'
      AND "group"='premium') <> 3 THEN
    RAISE EXCEPTION 'expected three current GPT channels';
  END IF;
  IF (SELECT count(DISTINCT key) FROM channels WHERE id IN (11,35,36)) <> 3 THEN
    RAISE EXCEPTION 'GPT channel keys are not distinct';
  END IF;
  IF EXISTS (SELECT 1 FROM channels WHERE name LIKE 'junxing-gpt-6-sol-legacy-host-%') THEN
    RAISE EXCEPTION 'legacy-host channels already exist';
  END IF;
END $$;

CREATE TABLE token123_change_log.token123_junxing_gpt6_legacy_channels_backup_20260928 AS
  SELECT * FROM channels WHERE id IN (11,35,36);
CREATE TABLE token123_change_log.token123_junxing_gpt6_legacy_abilities_backup_20260928 AS
  SELECT * FROM abilities WHERE channel_id IN (11,35,36) AND model='gpt-6-sol';
CREATE TABLE token123_change_log.token123_junxing_gpt6_legacy_dynamic_backup_20260928 AS
  SELECT * FROM options WHERE key='DynamicTextPricing';
CREATE TABLE token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928 (
  id bigint PRIMARY KEY,
  source_id bigint NOT NULL UNIQUE
);

UPDATE channels SET models='gpt-5.6-sol,gpt-6-astra,gpt-5.6-terra' WHERE id IN (11,35,36);
DELETE FROM abilities WHERE channel_id IN (11,35,36) AND model='gpt-6-sol';

WITH inserted AS (
  INSERT INTO channels (
    type,key,open_ai_organization,test_model,status,name,weight,created_time,test_time,response_time,
    base_url,other,balance,balance_updated_time,models,"group",used_quota,model_mapping,
    status_code_mapping,priority,auto_ban,other_info,tag,setting,param_override,header_override,
    remark,channel_info,settings
  )
  SELECT c.type,c.key,c.open_ai_organization,c.test_model,1,
    'junxing-gpt-6-sol-legacy-host-' || c.id,c.weight,extract(epoch FROM now())::bigint,
    NULL,NULL,'https://ai.centos.hk',c.other,c.balance,c.balance_updated_time,
    'gpt-6-sol','premium',0,c.model_mapping,c.status_code_mapping,c.priority,c.auto_ban,
    c.other_info,c.tag,c.setting,c.param_override,c.header_override,
    'gpt-pro 0.4x; gpt-6-sol only; upstream api host 503; 2.2x resale',c.channel_info,c.settings
  FROM token123_change_log.token123_junxing_gpt6_legacy_channels_backup_20260928 c
  RETURNING id,name
)
INSERT INTO token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928 (id,source_id)
SELECT i.id,substring(i.name FROM '([0-9]+)$')::bigint FROM inserted i;

INSERT INTO abilities ("group",model,channel_id,enabled,priority,weight,tag)
SELECT 'premium','gpt-6-sol',c.id,true,c.priority,c.weight,c.tag
FROM channels c
JOIN token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928 n ON n.id=c.id;

UPDATE options SET value=(
  SELECT jsonb_agg(
    CASE WHEN rule->>'name'='junxing-gpt-pro-long-context'
      THEN jsonb_set(rule,'{channel_ids}',
        '[11,35,36]'::jsonb ||
        (SELECT jsonb_agg(id ORDER BY id) FROM token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928))
      ELSE rule END ORDER BY ord)::text
  FROM jsonb_array_elements(value::jsonb) WITH ORDINALITY AS r(rule,ord)
) WHERE key='DynamicTextPricing';

DO $$
BEGIN
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928) <> 3 THEN
    RAISE EXCEPTION 'expected three new GPT 6 Sol channels';
  END IF;
  IF (SELECT count(DISTINCT c.key) FROM channels c
      JOIN token123_change_log.token123_junxing_gpt6_legacy_new_channels_20260928 n ON n.id=c.id) <> 3 THEN
    RAISE EXCEPTION 'new GPT 6 Sol keys are not distinct';
  END IF;
  IF (SELECT count(*) FROM abilities WHERE enabled AND "group"='premium' AND model='gpt-6-sol') <> 3 THEN
    RAISE EXCEPTION 'GPT 6 Sol route count is not three';
  END IF;
  IF (SELECT count(*) FROM abilities WHERE enabled AND channel_id IN (11,35,36) AND model='gpt-6-sol') <> 0 THEN
    RAISE EXCEPTION 'old host routes remain enabled for GPT 6 Sol';
  END IF;
  IF (SELECT jsonb_array_length(rule->'channel_ids') FROM options,
      LATERAL jsonb_array_elements(value::jsonb) AS r(rule)
      WHERE key='DynamicTextPricing' AND rule->>'name'='junxing-gpt-pro-long-context') <> 6 THEN
    RAISE EXCEPTION 'GPT long-context rule needs all six channel IDs';
  END IF;
END $$;
COMMIT;
