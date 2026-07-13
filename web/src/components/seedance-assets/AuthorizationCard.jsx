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
import React, { useEffect, useMemo, useState } from 'react';
import { Banner, Button, Card, Space, Typography } from '@douyinfe/semi-ui';
import {
  AlertCircle,
  CheckCircle2,
  Clock3,
  ExternalLink,
  ShieldCheck,
} from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { AUTH_STATE } from './state';

const { Paragraph, Text, Title } = Typography;

const authorizationStatus = (state, t) => {
  const states = {
    [AUTH_STATE.CREATING]: [<Clock3 size={18} />, t('正在创建授权邀请')],
    [AUTH_STATE.WAITING]: [<Clock3 size={18} />, t('等待演员本人完成授权')],
    [AUTH_STATE.EXPIRED]: [
      <AlertCircle size={18} />,
      t('授权邀请已过期，请重新生成。'),
    ],
    [AUTH_STATE.FAILED]: [<AlertCircle size={18} />, t('授权未完成')],
    [AUTH_STATE.AUTHORIZED]: [<CheckCircle2 size={18} />, t('真人授权已完成')],
  };
  return states[state] || [<ShieldCheck size={18} />, t('等待开始授权')];
};

const AuthorizationCard = ({
  state,
  actor,
  handoffUrl,
  expiresAt,
  errorMessage,
  onStart,
  onRetry,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!expiresAt || state !== AUTH_STATE.WAITING) return undefined;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [expiresAt, state]);

  const remainingSeconds = Math.max(
    0,
    Math.ceil((expiresAt * 1000 - now) / 1000),
  );
  const status = useMemo(() => authorizationStatus(state, t), [state, t]);

  if (state === AUTH_STATE.UNCONFIGURED) {
    return (
      <Card title={t('演员本人活体核验与肖像授权')}>
        <Banner
          type='warning'
          description={t('真人素材库尚未配置，请联系管理员。')}
        />
      </Card>
    );
  }

  return (
    <Card title={t('演员本人活体核验与肖像授权')}>
      <Space vertical align='start' spacing='medium' className='w-full'>
        <div className='flex items-center gap-2' aria-live='polite'>
          {status[0]}
          <Text strong>{status[1]}</Text>
        </div>
        <Paragraph>
          {t('当前演员：{{name}}', { name: actor?.display_name || '—' })}
        </Paragraph>
        <Paragraph type='tertiary'>
          {t(
            '二维码会先打开 Token123 授权说明页；演员本人确认期限和用途后，才会进入 BytePlus 活体核验。',
          )}
        </Paragraph>

        {(state === AUTH_STATE.UNAUTHORIZED ||
          state === AUTH_STATE.CHECKING) && (
          <Button
            type='primary'
            theme='solid'
            disabled={state === AUTH_STATE.CHECKING}
            onClick={onStart}
          >
            {t('生成演员授权二维码')}
          </Button>
        )}

        {state === AUTH_STATE.CREATING && (
          <Button type='primary' theme='solid' loading disabled>
            {t('正在创建授权邀请')}
          </Button>
        )}

        {state === AUTH_STATE.WAITING && (
          <div className='w-full'>
            <Banner
              type='info'
              description={t(
                '请让对应演员使用有摄像头的手机扫码；原设备会自动等待结果。',
              )}
            />
            {errorMessage && (
              <Banner
                className='mt-3'
                type='warning'
                description={errorMessage}
              />
            )}
            {handoffUrl ? (
              <div className='mt-4 flex flex-col items-center gap-4 md:items-start'>
                {!isMobile && (
                  <div
                    className='rounded-xl bg-white p-3 shadow-sm'
                    aria-label={t('演员手机授权二维码')}
                  >
                    <QRCodeSVG value={handoffUrl} size={208} level='M' />
                  </div>
                )}
                <Text>
                  {t('授权邀请剩余 {{seconds}} 秒', {
                    seconds: remainingSeconds,
                  })}
                </Text>
                <Button
                  type='primary'
                  theme='solid'
                  icon={<ExternalLink size={16} />}
                  onClick={() => window.location.assign(handoffUrl)}
                >
                  {isMobile
                    ? t('在本机交给演员确认')
                    : t('在当前设备打开交接页')}
                </Button>
              </div>
            ) : (
              <Button className='mt-3' onClick={onRetry}>
                {t('重新生成授权二维码')}
              </Button>
            )}
          </div>
        )}

        {(state === AUTH_STATE.EXPIRED || state === AUTH_STATE.FAILED) && (
          <>
            {errorMessage && (
              <Banner type='danger' description={errorMessage} />
            )}
            <Button type='primary' theme='solid' onClick={onRetry}>
              {t('重新生成授权二维码')}
            </Button>
          </>
        )}

        {state === AUTH_STATE.AUTHORIZED && (
          <Banner type='success' description={t('真人授权已完成')} />
        )}

        <Title heading={6}>{t('重要说明')}</Title>
        <Text type='tertiary'>
          {t('授权必须由对应演员本人主动完成，不能跳过或由他人代办。')}
        </Text>
      </Space>
    </Card>
  );
};

export default AuthorizationCard;
