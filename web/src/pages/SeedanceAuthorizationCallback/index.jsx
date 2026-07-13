/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import React, { useMemo } from 'react';
import { Banner, Button, Card, Space, Typography } from '@douyinfe/semi-ui';
import { CheckCircle2, Copy } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { copy, showError, showSuccess } from '../../helpers';

const { Paragraph, Title } = Typography;
const REVOCATION_TOKEN_KEY = 'seedance_actor_revocation_token';
const REVOCATION_ACTOR_KEY = 'seedance_actor_revocation_actor';

const SeedanceAuthorizationCallback = () => {
  const { t } = useTranslation();
  const revokeUrl = useMemo(() => {
    const token = sessionStorage.getItem(REVOCATION_TOKEN_KEY) || '';
    const actor = sessionStorage.getItem(REVOCATION_ACTOR_KEY) || '';
    if (!token) return '';
    const fragment = new URLSearchParams({ token, actor });
    return `${window.location.origin}/seedance/authorization/revoke#${fragment}`;
  }, []);

  const copyRevokeUrl = async () => {
    if (await copy(revokeUrl)) {
      showSuccess(t('撤回链接已复制，请由演员本人妥善保存。'));
    } else {
      showError(t('复制失败，请手动保存撤回链接。'));
    }
  };

  return (
    <main className='mx-auto flex min-h-screen max-w-xl items-center px-4 py-10'>
      <Card className='w-full text-center'>
        <Space vertical spacing='medium' align='center'>
          <CheckCircle2
            size={48}
            color='var(--semi-color-success)'
            aria-hidden='true'
          />
          <Title heading={2}>{t('授权完成，请返回原设备。')}</Title>
          <Paragraph>
            {t(
              '手机端无需登录 Token123。原设备会自动查询授权结果并进入下一步。',
            )}
          </Paragraph>
          {revokeUrl && (
            <Banner
              className='text-left'
              type='warning'
              title={t('请保存演员撤回链接')}
              description={t(
                '该链接允许演员本人随时撤回授权。请勿交给无关人员，也不要发布到公开渠道。',
              )}
            />
          )}
          {revokeUrl && (
            <Button icon={<Copy size={15} />} onClick={copyRevokeUrl}>
              {t('复制撤回链接')}
            </Button>
          )}
          <Paragraph type='tertiary'>
            {t('请关闭本页面并返回原设备。')}
          </Paragraph>
        </Space>
      </Card>
    </main>
  );
};

export default SeedanceAuthorizationCallback;
