-- Token123 / 君的星辰：9 个已具备上游启用令牌的模型。
-- Claude Opus 5.5 单独在 Claude MAX 令牌获准启用后上架；本文件不提前暴露不可路由的模型。
-- 上游人民币单价 × 上游分组倍率 × 2.2 为售价。GPT 使用 gpt-pro 0.4x；
-- Claude Fable/Opus 5/Sonnet 5 使用 Claude特惠，图像使用生图分组。
-- Token123 premium=0.5、QuotaPerUnit=500000，故文本 ModelRatio 等于每百万输入 token 的最终售价；
-- 图片 ModelPrice=每次最终售价/0.5。无 USD/RMB 二次换算。
-- 执行前核实：目标数据库为 Token123 生产 PostgreSQL；生产已有动态文本计费版本和
-- GroupRatio.premium=0.5；渠道 10/32/33、11/35/36、43 的上游令牌真实列出对应模型。
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

DO $$
BEGIN
  IF (SELECT value::jsonb ->> 'premium' FROM options WHERE key='GroupRatio') IS DISTINCT FROM '0.5' THEN
    RAISE EXCEPTION 'premium group ratio is not 0.5';
  END IF;
  IF (SELECT count(*) FROM channels WHERE id IN (10,11,32,33,35,36,43) AND status=1) <> 7 THEN
    RAISE EXCEPTION 'expected seven active Junxing channels';
  END IF;
  IF (SELECT count(*) FROM options WHERE key IN ('ModelRatio','CompletionRatio','CacheRatio','CreateCacheRatio','ModelPrice','DynamicTextPricing')) <> 6 THEN
    RAISE EXCEPTION 'required pricing option is absent';
  END IF;
  IF EXISTS (
    SELECT model_name FROM models
    WHERE model_name IN ('gpt-5.6-sol','gpt-6-astra','gpt-5.6-terra','gpt-6-sol',
      'claude-fable-5-1','claude-opus-5','claude-sonnet-5',
      'gpt-image-2.5-flare','gpt-image-2.5-sunburst')
    GROUP BY model_name HAVING count(*) > 1
  ) THEN
    RAISE EXCEPTION 'duplicate target model rows require manual review';
  END IF;
END $$;

CREATE SCHEMA IF NOT EXISTS token123_change_log;
CREATE TABLE token123_change_log.token123_junxing_models_channels_backup_20260928 AS
  SELECT * FROM channels WHERE id IN (1,2,10,11,16,17,23,24,25,26,27,28,32,33,35,36,37,38,39,43);
CREATE TABLE token123_change_log.token123_junxing_models_abilities_backup_20260928 AS
  SELECT * FROM abilities WHERE channel_id IN (1,2,10,11,16,17,23,24,25,26,27,28,32,33,35,36,37,38,39,43)
     OR model IN (SELECT model_name FROM models WHERE vendor_id IN (1,2));
CREATE TABLE token123_change_log.token123_junxing_models_catalog_backup_20260928 AS
  SELECT * FROM models WHERE vendor_id IN (1,2);
CREATE TABLE token123_change_log.token123_junxing_models_options_backup_20260928 AS
  SELECT * FROM options WHERE key IN ('ModelRatio','CompletionRatio','CacheRatio','CreateCacheRatio','ModelPrice','DynamicTextPricing');

-- 退役旧 OneRouter/OpenAI/Anthropic 渠道与君的星辰旧 Codex Luna 渠道。
UPDATE channels SET status=2 WHERE id IN (1,2,16,17,23,24,25,26,27,28,37,38,39);
-- 当前全部目标模型只售 premium；不允许旧 default/balanced 绕过这一售价。
UPDATE channels SET "group"='premium', models='gpt-5.6-sol,gpt-6-astra,gpt-5.6-terra,gpt-6-sol', base_url='https://api.centos.hk'
  WHERE id IN (11,35,36);
UPDATE channels SET "group"='premium', models='claude-fable-5-1,claude-opus-5,claude-sonnet-5', base_url='https://api.centos.hk'
  WHERE id IN (10,32,33);
UPDATE channels SET "group"='premium', models='gpt-image-2.5-flare,gpt-image-2.5-sunburst', base_url='https://api.centos.hk'
  WHERE id=43;

-- 能力表与渠道模型列表同步，含禁用旧渠道。其它供应商的能力不变。
DELETE FROM abilities WHERE channel_id IN (1,2,10,11,16,17,23,24,25,26,27,28,32,33,35,36,37,38,39,43);
INSERT INTO abilities ("group",model,channel_id,enabled,priority,weight,tag)
SELECT trim(g),trim(m),c.id,c.status=1,c.priority,c.weight,c.tag
FROM channels c
CROSS JOIN LATERAL unnest(string_to_array(c."group",',')) g
CROSS JOIN LATERAL unnest(string_to_array(c.models,',')) m
WHERE c.id IN (1,2,10,11,16,17,23,24,25,26,27,28,32,33,35,36,37,38,39,43)
ON CONFLICT ("group",model,channel_id) DO UPDATE SET enabled=EXCLUDED.enabled,priority=EXCLUDED.priority,weight=EXCLUDED.weight,tag=EXCLUDED.tag;
UPDATE abilities SET enabled=false
WHERE model IN (SELECT model_name FROM models WHERE vendor_id IN (1,2))
  AND model NOT IN ('gpt-5.6-sol','gpt-6-astra','gpt-5.6-terra','gpt-6-sol','claude-fable-5-1','claude-opus-5','claude-sonnet-5','gpt-image-2.5-flare','gpt-image-2.5-sunburst');

-- 目录仅展示实际可路由的 9 款。现有 Opus 5、Sonnet 5、Sol、Terra 保留原 ID。
UPDATE models SET status=0,updated_time=extract(epoch FROM now())::bigint
WHERE vendor_id IN (1,2)
  AND model_name NOT IN ('gpt-5.6-sol','gpt-6-astra','gpt-5.6-terra','gpt-6-sol','claude-fable-5-1','claude-opus-5','claude-sonnet-5','gpt-image-2.5-flare','gpt-image-2.5-sunburst');
INSERT INTO models (model_name,description,icon,tags,vendor_id,endpoints,status,sync_official,created_time,updated_time,name_rule)
SELECT v.model_name,v.description,v.icon,v.tags,v.vendor_id,'',1,0,extract(epoch FROM now())::bigint,extract(epoch FROM now())::bigint,0
FROM (VALUES
 ('gpt-5.6-sol','GPT-5.6 Sol','OpenAI','Reasoning,Tools,Files,Vision',2),
 ('gpt-6-astra','GPT-6 Astra','OpenAI','Reasoning,Tools,Files,Vision',2),
 ('gpt-5.6-terra','GPT-5.6 Terra','OpenAI','Reasoning,Tools,Files,Vision',2),
 ('gpt-6-sol','GPT-6 Sol','OpenAI','Reasoning,Tools,Files,Vision',2),
 ('claude-fable-5-1','Claude Fable 5.1','Claude.Color','Reasoning,Tools,Files,Vision',1),
 ('claude-opus-5','Claude Opus 5','Claude.Color','Reasoning,Tools,Files,Vision',1),
 ('claude-sonnet-5','Claude Sonnet 5','Claude.Color','Reasoning,Tools,Files,Vision',1),
 ('gpt-image-2.5-flare','GPT Image 2.5 Flare','OpenAI','Image,Vision',2),
 ('gpt-image-2.5-sunburst','GPT Image 2.5 Sunburst','OpenAI','Image,Vision',2)
) AS v(model_name,description,icon,tags,vendor_id)
WHERE NOT EXISTS (SELECT 1 FROM models m WHERE m.model_name=v.model_name);
UPDATE models SET status=1,deleted_at=NULL,updated_time=extract(epoch FROM now())::bigint
WHERE model_name IN ('gpt-5.6-sol','gpt-6-astra','gpt-5.6-terra','gpt-6-sol','claude-fable-5-1','claude-opus-5','claude-sonnet-5','gpt-image-2.5-flare','gpt-image-2.5-sunburst');

UPDATE options SET value=(value::jsonb || '{"gpt-5.6-sol":4.4,"gpt-6-astra":8.8,"gpt-5.6-terra":1.76,"gpt-6-sol":1.76,"claude-fable-5-1":2.2,"claude-opus-5":1.1,"claude-sonnet-5":0.66}'::jsonb)::text WHERE key='ModelRatio';
UPDATE options SET value=(value::jsonb || '{"gpt-5.6-sol":6,"gpt-6-astra":5,"gpt-5.6-terra":6,"gpt-6-sol":5,"claude-fable-5-1":5,"claude-opus-5":5,"claude-sonnet-5":5}'::jsonb)::text WHERE key='CompletionRatio';
UPDATE options SET value=(value::jsonb || '{"gpt-5.6-sol":0.1,"gpt-6-astra":0.1,"gpt-5.6-terra":0.1,"gpt-6-sol":0.1,"claude-fable-5-1":0.025,"claude-opus-5":0.1,"claude-sonnet-5":0.1}'::jsonb)::text WHERE key='CacheRatio';
UPDATE options SET value=(value::jsonb || '{"gpt-5.6-sol":1.25,"gpt-6-astra":1.25,"gpt-5.6-terra":1.25,"gpt-6-sol":1.25,"claude-fable-5-1":2,"claude-opus-5":2,"claude-sonnet-5":1.25}'::jsonb)::text WHERE key='CreateCacheRatio';
UPDATE options SET value=(value::jsonb || '{"gpt-image-2.5-flare":0.528,"gpt-image-2.5-sunburst":0.66}'::jsonb)::text WHERE key='ModelPrice';
UPDATE options SET value=(value::jsonb || '[{"name":"junxing-gpt-pro-long-context","groups":["premium"],"models":["gpt-5.6-sol","gpt-6-astra","gpt-5.6-terra","gpt-6-sol"],"channel_ids":[11,35,36],"min_prompt_tokens":272001,"input_multiplier":2,"output_multiplier":1.5,"cache_read_multiplier":2,"cache_write_multiplier":2}]'::jsonb)::text WHERE key='DynamicTextPricing';

DO $$
BEGIN
  IF (SELECT count(*) FROM models WHERE vendor_id IN (1,2) AND status=1) <> 9 THEN
    RAISE EXCEPTION 'enabled OpenAI/Anthropic catalog count must be nine before Claude MAX';
  END IF;
  IF (SELECT count(*) FROM channels WHERE id IN (10,11,32,33,35,36,43) AND status=1 AND "group"='premium') <> 7 THEN
    RAISE EXCEPTION 'seven active channels must use premium';
  END IF;
  IF (SELECT count(*) FROM abilities WHERE enabled AND channel_id IN (10,11,32,33,35,36,43)) <> 23 THEN
    RAISE EXCEPTION 'expected 23 enabled routing abilities';
  END IF;
END $$;
COMMIT;
