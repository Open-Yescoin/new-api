import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import {
  AlertCircle,
  CheckCircle2,
  Clock3,
  ExternalLink,
  ShieldCheck,
} from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';
import { useTranslation } from 'react-i18next';
import { copy, showError, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { AUTH_STATE } from './state';

const { Paragraph, Text, Title } = Typography;

const authorizationStatus = (state, t) => {
  switch (state) {
    case AUTH_STATE.CREATING:
      return { icon: <Clock3 size={18} />, text: t('正在创建授权链接') };
    case AUTH_STATE.WAITING:
      return { icon: <Clock3 size={18} />, text: t('等待本人完成授权') };
    case AUTH_STATE.EXPIRED:
      return {
        icon: <AlertCircle size={18} />,
        text: t('授权链接已过期，请重新生成。'),
      };
    case AUTH_STATE.FAILED:
      return { icon: <AlertCircle size={18} />, text: t('授权未完成') };
    case AUTH_STATE.AUTHORIZED:
      return { icon: <CheckCircle2 size={18} />, text: t('真人授权已完成') };
    default:
      return { icon: <ShieldCheck size={18} />, text: t('等待开始授权') };
  }
};

const AuthorizationCard = ({
  state,
  consented,
  onConsentChange,
  h5Link,
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

  const copyAuthorizationLink = async () => {
    if (!h5Link) return;
    if (await copy(h5Link)) {
      showSuccess(t('授权链接已复制'));
    } else {
      showError(t('复制失败，请手动打开授权链接'));
    }
  };

  if (state === AUTH_STATE.UNCONFIGURED) {
    return (
      <Card title={t('本人活体核验与肖像授权')}>
        <Banner
          type='warning'
          description={t('真人素材库尚未配置，请联系管理员。')}
        />
      </Card>
    );
  }

  return (
    <Card title={t('本人活体核验与肖像授权')}>
      <Space vertical align='start' spacing='medium' className='w-full'>
        <div className='flex items-center gap-2' aria-live='polite'>
          {status.icon}
          <Text strong>{status.text}</Text>
        </div>
        <Paragraph>
          {t(
            '授权用于核验本人、建立真人私有素材组，并检查后续素材与本人一致。',
          )}
        </Paragraph>
        <Paragraph type='tertiary'>
          {t('若所有可用设备都没有摄像头，将无法完成真人授权。')}
        </Paragraph>

        {(state === AUTH_STATE.UNAUTHORIZED ||
          state === AUTH_STATE.CHECKING) && (
          <>
            <Checkbox
              checked={consented}
              onChange={(event) => onConsentChange(event.target.checked)}
            >
              {t(
                '我确认由本人完成活体核验，并授权平台为 Seedance 真人素材生成处理我的人脸与肖像信息。',
              )}
            </Checkbox>
            <Button
              type='primary'
              theme='solid'
              disabled={!consented || state === AUTH_STATE.CHECKING}
              onClick={onStart}
            >
              {t('开始本人授权')}
            </Button>
          </>
        )}

        {state === AUTH_STATE.CREATING && (
          <Button type='primary' theme='solid' loading disabled>
            {t('正在创建授权链接')}
          </Button>
        )}

        {state === AUTH_STATE.WAITING && (
          <div className='w-full'>
            <Banner
              type='info'
              description={t(
                '请使用有摄像头的手机扫码完成授权；原设备会自动等待结果。',
              )}
            />
            {h5Link ? (
              <div className='mt-4 flex flex-col items-center gap-4 md:items-start'>
                {!isMobile && (
                  <div
                    className='rounded-xl bg-white p-3 shadow-sm'
                    aria-label={t('手机授权二维码')}
                  >
                    <QRCodeSVG value={h5Link} size={208} level='M' />
                  </div>
                )}
                <Text>
                  {t('授权链接剩余 {{seconds}} 秒', {
                    seconds: remainingSeconds,
                  })}
                </Text>
                <div className='max-w-full break-all rounded-lg bg-[var(--semi-color-fill-0)] p-3 text-sm'>
                  {h5Link}
                </div>
                <Space wrap>
                  <Button onClick={copyAuthorizationLink}>
                    {t('复制授权链接')}
                  </Button>
                  <Button
                    type='primary'
                    theme='solid'
                    icon={<ExternalLink size={16} />}
                    onClick={() => window.location.assign(h5Link)}
                  >
                    {isMobile ? t('在本机继续授权') : t('在当前设备打开')}
                  </Button>
                </Space>
              </div>
            ) : (
              <div className='mt-4'>
                <Banner
                  type='warning'
                  description={t(
                    '此设备没有原授权链接，请返回发起授权的设备，或重新生成。',
                  )}
                />
                <Button className='mt-3' onClick={onRetry}>
                  {t('重新生成授权链接')}
                </Button>
              </div>
            )}
          </div>
        )}

        {(state === AUTH_STATE.EXPIRED || state === AUTH_STATE.FAILED) && (
          <>
            {errorMessage && (
              <Banner type='danger' description={errorMessage} />
            )}
            <Button type='primary' theme='solid' onClick={onRetry}>
              {t('重新生成授权链接')}
            </Button>
          </>
        )}

        {state === AUTH_STATE.AUTHORIZED && (
          <Banner type='success' description={t('真人授权已完成')} />
        )}

        <Title heading={6}>{t('重要说明')}</Title>
        <Text type='tertiary'>
          {t('授权必须由真人本人主动完成，不能跳过或由他人代办。')}
        </Text>
      </Space>
    </Card>
  );
};

export default AuthorizationCard;
