-- Token123 生产：将君的星辰 GPT/Claude 文本渠道切至上游建议的直连地址。
-- ai.centos.hk/pricing 是价目页；上游官方 FAQ 针对 Codex/Claude 524 推荐 api.centos.hk。
-- 执行前提：目标 6 渠道仍启用且 base_url 均为 https://ai.centos.hk。
-- 备份只包含本次会改动的字段，不复制密钥。

\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '15s';
CREATE SCHEMA IF NOT EXISTS token123_change_log;

CREATE TABLE token123_change_log.token123_junxing_524_url_backup_20260928 AS
SELECT id, base_url, status
FROM public.channels
WHERE id IN (10, 11, 32, 33, 35, 36);

DO $$
BEGIN
  IF (SELECT count(*) FROM token123_change_log.token123_junxing_524_url_backup_20260928
      WHERE base_url = 'https://ai.centos.hk' AND status = 1) <> 6 THEN
    RAISE EXCEPTION 'source channels changed; stop before URL update';
  END IF;
END $$;

UPDATE public.channels
SET base_url = 'https://api.centos.hk'
WHERE id IN (10, 11, 32, 33, 35, 36)
  AND base_url = 'https://ai.centos.hk'
  AND status = 1;

DO $$
BEGIN
  IF (SELECT count(*) FROM public.channels
      WHERE id IN (10, 11, 32, 33, 35, 36)
        AND base_url = 'https://api.centos.hk' AND status = 1) <> 6 THEN
    RAISE EXCEPTION 'destination channel count differs from 6';
  END IF;
END $$;

COMMIT;

-- 回滚（单独执行）：
-- BEGIN;
-- UPDATE public.channels AS c SET base_url = b.base_url
-- FROM token123_change_log.token123_junxing_524_url_backup_20260928 AS b
-- WHERE c.id = b.id AND c.base_url = 'https://api.centos.hk';
-- COMMIT;
