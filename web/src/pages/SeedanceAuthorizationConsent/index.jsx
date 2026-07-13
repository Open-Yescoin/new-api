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
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Space,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { ShieldCheck } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { seedanceAssetsApi } from '../../services/seedanceAssets';

const { Paragraph, Text, Title } = Typography;
export const SEEDANCE_REVOCATION_TOKEN_KEY = 'seedance_actor_revocation_token';
export const SEEDANCE_REVOCATION_ACTOR_KEY = 'seedance_actor_revocation_actor';

const durationLabel = (durationDays, t) => {
  if (durationDays === null || durationDays === undefined) {
    return t('长期有效（演员可随时撤回）');
  }
  if (durationDays === 180) return t('半年授权');
  if (durationDays === 365) return t('一年授权');
  return t('{{days}} 天授权', { days: durationDays });
};

const SeedanceAuthorizationConsent = () => {
  const { t } = useTranslation();
  const [handoff, setHandoff] = useState(null);
  const [details, setDetails] = useState(null);
  const [consented, setConsented] = useState(false);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const fragment = new URLSearchParams(window.location.hash.slice(1));
    const invitationToken = fragment.get('invitation') || '';
    const h5Link = fragment.get('h5') || '';
    const revocationToken = fragment.get('revoke') || '';
    window.history.replaceState(null, '', window.location.pathname);

    let parsedH5;
    try {
      parsedH5 = new URL(h5Link);
    } catch (_error) {
      setError(t('授权邀请无效或已经过期。'));
      setLoading(false);
      return;
    }
    if (
      !invitationToken ||
      !revocationToken ||
      parsedH5.protocol !== 'https:'
    ) {
      setError(t('授权邀请无效或已经过期。'));
      setLoading(false);
      return;
    }
    const nextHandoff = { invitationToken, h5Link, revocationToken };
    setHandoff(nextHandoff);
    seedanceAssetsApi
      .publicInvitationDetails(invitationToken)
      .then((response) => setDetails(response.data?.data || null))
      .catch(() => setError(t('授权邀请无效或已经过期。')))
      .finally(() => setLoading(false));
  }, [t]);

  const acceptAndContinue = async () => {
    if (!consented || !handoff || !details) return;
    setSubmitting(true);
    setError('');
    try {
      await seedanceAssetsApi.publicAcceptConsent(handoff.invitationToken);
      sessionStorage.setItem(
        SEEDANCE_REVOCATION_TOKEN_KEY,
        handoff.revocationToken,
      );
      sessionStorage.setItem(
        SEEDANCE_REVOCATION_ACTOR_KEY,
        details.actor_name || '',
      );
      const h5Link = handoff.h5Link;
      window.location.assign(h5Link);
    } catch (_error) {
      setError(t('授权确认失败，请让邀请方重新生成二维码。'));
      setSubmitting(false);
    }
  };

  return (
    <main className='mx-auto flex min-h-screen max-w-xl items-center px-4 py-10'>
      <Card className='w-full'>
        <Space vertical spacing='medium' align='start' className='w-full'>
          <ShieldCheck
            size={44}
            color='var(--semi-color-primary)'
            aria-hidden='true'
          />
          <Title heading={2}>{t('Seedance 真人素材授权')}</Title>
          {loading ? (
            <Spin tip={t('正在读取授权邀请')} />
          ) : error ? (
            <Banner type='danger' description={error} />
          ) : (
            <>
              <Banner
                type='info'
                description={t(
                  '确认后将进入 BytePlus 本人活体核验；完成后邀请方才能登记与你一致的真人素材。',
                )}
              />
              <div className='w-full rounded-lg bg-[var(--semi-color-fill-0)] p-4'>
                <Paragraph>
                  <Text strong>{t('演员')}</Text>：{details?.actor_name}
                </Paragraph>
                <Paragraph>
                  <Text strong>{t('授权期限')}</Text>：
                  {durationLabel(details?.duration_days, t)}
                </Paragraph>
                <Paragraph>
                  <Text strong>{t('处理目的')}</Text>：
                  {t(
                    '核验本人、建立独立真人素材组，并用于你授权范围内的 Seedance 视频生成。',
                  )}
                </Paragraph>
                <Paragraph>
                  <Text strong>{t('撤回方式')}</Text>：
                  {t('完成后请保存撤回链接；撤回后平台会立即停止继续调用。')}
                </Paragraph>
              </div>
              <Checkbox
                checked={consented}
                onChange={(event) => setConsented(event.target.checked)}
              >
                {t(
                  '我确认由本人完成活体核验，并同意按上述期限和用途处理我的人脸与肖像素材。',
                )}
              </Checkbox>
              <Button
                type='primary'
                theme='solid'
                disabled={!consented}
                loading={submitting}
                onClick={acceptAndContinue}
              >
                {t('确认并开始本人核验')}
              </Button>
            </>
          )}
        </Space>
      </Card>
    </main>
  );
};

export default SeedanceAuthorizationConsent;
