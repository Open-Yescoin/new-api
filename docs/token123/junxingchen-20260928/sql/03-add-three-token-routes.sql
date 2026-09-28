-- Token123 / 君的星辰：为 Claude MAX 和生图分组补齐各三枚独立上游令牌。
-- 调用方通过 psql 变量 max_key_1..3、image_key_2..3 安全传入密钥；本文件不含密钥。
-- Opus 5.5 使用 Claude MAX 1.8x；其余 Claude 使用 Claude特惠 0.1x；图像使用生图/视频 1x。
-- 所有上游人民币成本乘 2.2 售卖。premium=0.5，文本 ModelRatio 为每百万输入 token 最终价。
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

DO $$
BEGIN
  IF (SELECT count(*) FROM channels WHERE id IN (10,43) AND status=1 AND "group"='premium') <> 2 THEN
    RAISE EXCEPTION 'expected Claude and image source channels';
  END IF;
  IF (SELECT count(*) FROM models WHERE vendor_id IN (1,2) AND status=1) <> 9 THEN
    RAISE EXCEPTION 'expected nine models before Opus 5.5';
  END IF;
  IF EXISTS (SELECT 1 FROM models WHERE model_name='claude-opus-5-5') THEN
    RAISE EXCEPTION 'Opus 5.5 model row already exists; review it before inserting';
  END IF;
  IF EXISTS (SELECT 1 FROM channels WHERE name IN ('junxing-claude-opus55-1','junxing-claude-opus55-2','junxing-claude-opus55-3','junxing-gpt-image-25-2','junxing-gpt-image-25-3')) THEN
    RAISE EXCEPTION 'new channel names already exist';
  END IF;
END $$;

CREATE TABLE token123_change_log.token123_junxing_three_token_options_backup_20260928 AS
  SELECT * FROM options WHERE key IN ('ModelRatio','CompletionRatio','CacheRatio','CreateCacheRatio','DynamicTextPricing');
CREATE TABLE token123_change_log.token123_junxing_three_token_channels_20260928 (
  id bigint PRIMARY KEY,
  name text NOT NULL
);

-- 克隆已验真的同类渠道设置，分别使用独立密钥。只转发真实客户端 User-Agent；
-- Claude MAX 上游要求真实 Claude Code 客户端，普通 HTTP 客户端会被上游拒绝。
WITH inserted AS (
  INSERT INTO channels (
    type,key,open_ai_organization,test_model,status,name,weight,created_time,test_time,response_time,
    base_url,other,balance,balance_updated_time,models,"group",used_quota,model_mapping,
    status_code_mapping,priority,auto_ban,other_info,tag,setting,param_override,header_override,
    remark,channel_info,settings
  )
  SELECT c.type,v.key,c.open_ai_organization,c.test_model,1,v.name,c.weight,
    extract(epoch FROM now())::bigint,NULL,NULL,'https://api.centos.hk',c.other,c.balance,
    c.balance_updated_time,'claude-opus-5-5','premium',0,c.model_mapping,
    c.status_code_mapping,c.priority,c.auto_ban,c.other_info,c.tag,c.setting,c.param_override,
    '{"User-Agent":"{client_header:User-Agent}"}',
    'Claude MAX 1.8x; genuine Claude Code only; 2.2x resale',c.channel_info,c.settings
  FROM channels c
  CROSS JOIN (VALUES
    ('junxing-claude-opus55-1', :'max_key_1'),
    ('junxing-claude-opus55-2', :'max_key_2'),
    ('junxing-claude-opus55-3', :'max_key_3')
  ) AS v(name,key)
  WHERE c.id=10
  RETURNING id,name
)
INSERT INTO token123_change_log.token123_junxing_three_token_channels_20260928 SELECT * FROM inserted;

WITH inserted AS (
  INSERT INTO channels (
    type,key,open_ai_organization,test_model,status,name,weight,created_time,test_time,response_time,
    base_url,other,balance,balance_updated_time,models,"group",used_quota,model_mapping,
    status_code_mapping,priority,auto_ban,other_info,tag,setting,param_override,header_override,
    remark,channel_info,settings
  )
  SELECT c.type,v.key,c.open_ai_organization,c.test_model,1,v.name,c.weight,
    extract(epoch FROM now())::bigint,NULL,NULL,'https://api.centos.hk',c.other,c.balance,
    c.balance_updated_time,'gpt-image-2.5-flare,gpt-image-2.5-sunburst','premium',0,c.model_mapping,
    c.status_code_mapping,c.priority,c.auto_ban,c.other_info,c.tag,c.setting,c.param_override,
    c.header_override,'生图/视频分组; 2.2x resale',c.channel_info,c.settings
  FROM channels c
  CROSS JOIN (VALUES
    ('junxing-gpt-image-25-2', :'image_key_2'),
    ('junxing-gpt-image-25-3', :'image_key_3')
  ) AS v(name,key)
  WHERE c.id=43
  RETURNING id,name
)
INSERT INTO token123_change_log.token123_junxing_three_token_channels_20260928 SELECT * FROM inserted;

INSERT INTO abilities ("group",model,channel_id,enabled,priority,weight,tag)
SELECT trim(g),trim(m),c.id,true,c.priority,c.weight,c.tag
FROM channels c
JOIN token123_change_log.token123_junxing_three_token_channels_20260928 b ON b.id=c.id
CROSS JOIN LATERAL unnest(string_to_array(c."group",',')) g
CROSS JOIN LATERAL unnest(string_to_array(c.models,',')) m;

INSERT INTO models (model_name,description,icon,tags,vendor_id,endpoints,status,sync_official,created_time,updated_time,name_rule)
VALUES ('claude-opus-5-5','Claude Opus 5.5 · Claude Code 客户端专用','Claude.Color',
        'Reasoning,Tools,Files,Vision',1,'',1,0,extract(epoch FROM now())::bigint,
        extract(epoch FROM now())::bigint,0);
CREATE TABLE token123_change_log.token123_junxing_three_token_model_20260928 AS
  SELECT id FROM models WHERE model_name='claude-opus-5-5' AND vendor_id=1;

-- MAX 上游标准成本 ¥7.2/36/0.36/14.4；售价 ¥15.84/79.2/0.792/31.68。
UPDATE options SET value=(value::jsonb || '{"claude-opus-5-5":15.84}'::jsonb)::text WHERE key='ModelRatio';
UPDATE options SET value=(value::jsonb || '{"claude-opus-5-5":5}'::jsonb)::text WHERE key='CompletionRatio';
UPDATE options SET value=(value::jsonb || '{"claude-opus-5-5":0.05}'::jsonb)::text WHERE key='CacheRatio';
UPDATE options SET value=(value::jsonb || '{"claude-opus-5-5":2}'::jsonb)::text WHERE key='CreateCacheRatio';

-- 已退役的 OneRouter 档位与 Junxing 的模型名重叠，会污染公开价格页的档位名称。
-- 保留 Gemini 档位和本次 GPT 长上下文档位，移除已退役 OneRouter 档位。
UPDATE options SET value=(
  SELECT jsonb_agg(rule ORDER BY ord)::text
  FROM jsonb_array_elements(value::jsonb) WITH ORDINALITY AS r(rule,ord)
  WHERE rule->>'name' <> 'onerouter-openai-long-context'
) WHERE key='DynamicTextPricing';

DO $$
BEGIN
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_three_token_channels_20260928) <> 5 THEN
    RAISE EXCEPTION 'expected five new channels';
  END IF;
  IF (SELECT count(DISTINCT key) FROM channels WHERE id=43 OR id IN (SELECT id FROM token123_change_log.token123_junxing_three_token_channels_20260928)) <> 6 THEN
    RAISE EXCEPTION 'duplicate upstream key';
  END IF;
  IF (SELECT count(*) FROM abilities WHERE enabled AND channel_id IN (SELECT id FROM token123_change_log.token123_junxing_three_token_channels_20260928)) <> 7 THEN
    RAISE EXCEPTION 'expected seven new routing abilities';
  END IF;
  IF (SELECT count(*) FROM models WHERE vendor_id IN (1,2) AND status=1) <> 10 THEN
    RAISE EXCEPTION 'expected ten enabled models';
  END IF;
END $$;
COMMIT;
