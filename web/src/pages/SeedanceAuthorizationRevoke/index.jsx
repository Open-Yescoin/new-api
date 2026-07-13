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
import React, { useEffect, useState } from 'react';
import { Banner, Button, Card, Space, Typography } from '@douyinfe/semi-ui';
import { ShieldOff } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { seedanceAssetsApi } from '../../services/seedanceAssets';

const { Paragraph, Title } = Typography;

const SeedanceAuthorizationRevoke = () => {
  const { t } = useTranslation();
  const [token, setToken] = useState('');
  const [actorName, setActorName] = useState('');
  const [status, setStatus] = useState('ready');

  useEffect(() => {
    const fragment = new URLSearchParams(window.location.hash.slice(1));
    setToken(fragment.get('token') || '');
    setActorName(fragment.get('actor') || '');
    window.history.replaceState(null, '', window.location.pathname);
  }, []);

  const revoke = async () => {
    if (!token) return;
    setStatus('loading');
    try {
      await seedanceAssetsApi.publicRevoke(token);
      setStatus('revoked');
    } catch (_error) {
      setStatus('failed');
    }
  };

  return (
    <main className='mx-auto flex min-h-screen max-w-xl items-center px-4 py-10'>
      <Card className='w-full'>
        <Space vertical spacing='medium' align='start'>
          <ShieldOff size={44} color='var(--semi-color-danger)' />
          <Title heading={2}>{t('撤回 Seedance 真人素材授权')}</Title>
          <Paragraph>
            {t('演员：{{name}}', { name: actorName || '—' })}
          </Paragraph>
          <Paragraph>
            {t(
              '撤回后会立即停止平台新建和调用该演员的真人素材，不会影响其他演员。',
            )}
          </Paragraph>
          {!token && <Banner type='danger' description={t('撤回链接无效。')} />}
          {status === 'revoked' && (
            <Banner type='success' description={t('授权已撤回并立即停用。')} />
          )}
          {status === 'failed' && (
            <Banner
              type='danger'
              description={t('撤回失败，请联系平台支持。')}
            />
          )}
          {status !== 'revoked' && (
            <Button
              type='danger'
              theme='solid'
              disabled={!token}
              loading={status === 'loading'}
              onClick={revoke}
            >
              {t('立即撤回授权')}
            </Button>
          )}
        </Space>
      </Card>
    </main>
  );
};

export default SeedanceAuthorizationRevoke;
