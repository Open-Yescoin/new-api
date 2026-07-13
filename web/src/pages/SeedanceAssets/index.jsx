import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Banner, Card, Spin, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import AuthorizationCard from '../../components/seedance-assets/AuthorizationCard';
import AssetLibrary from '../../components/seedance-assets/AssetLibrary';
import {
  AUTH_STATE,
  authorizationStateFromPollError,
  authorizationStateFromStatus,
  shouldContinueAuthorizationPolling,
} from '../../components/seedance-assets/state';
import { seedanceAssetsApi } from '../../services/seedanceAssets';

const { Paragraph, Title } = Typography;
const AUTHORIZATION_TOKEN_KEY = 'seedance_authorization_token';
const responseErrorCode = (error) => error.response?.data?.error?.code;

const SeedanceAssets = () => {
  const { t } = useTranslation();
  const [authorizationState, setAuthorizationState] = useState(
    AUTH_STATE.CHECKING,
  );
  const [consented, setConsented] = useState(false);
  const [h5Link, setH5Link] = useState('');
  const [expiresAt, setExpiresAt] = useState(0);
  const [errorMessage, setErrorMessage] = useState('');
  const pollStartedAtRef = useRef(0);
  const pollInFlightRef = useRef(false);

  const clearAuthorizationSession = useCallback(() => {
    sessionStorage.removeItem(AUTHORIZATION_TOKEN_KEY);
    setH5Link('');
    setExpiresAt(0);
    pollStartedAtRef.current = 0;
  }, []);

  const loadAuthorizationStatus = useCallback(async () => {
    setAuthorizationState(AUTH_STATE.CHECKING);
    try {
      const response = await seedanceAssetsApi.authorizationStatus();
      const status = response.data?.data;
      const nextState = authorizationStateFromStatus(status);
      const token = sessionStorage.getItem(AUTHORIZATION_TOKEN_KEY);
      if (nextState === AUTH_STATE.AUTHORIZED) {
        clearAuthorizationSession();
        setAuthorizationState(AUTH_STATE.AUTHORIZED);
        return;
      }
      if (nextState === AUTH_STATE.WAITING && token) {
        pollStartedAtRef.current = Date.now();
        setAuthorizationState(AUTH_STATE.WAITING);
        return;
      }
      if (token) clearAuthorizationSession();
      setAuthorizationState(nextState);
    } catch (_error) {
      setErrorMessage(t('授权状态加载失败，请稍后重试。'));
      setAuthorizationState(AUTH_STATE.FAILED);
    }
  }, [clearAuthorizationSession, t]);

  useEffect(() => {
    loadAuthorizationStatus();
  }, [loadAuthorizationStatus]);

  const startAuthorization = useCallback(async () => {
    if (!consented) return;
    setAuthorizationState(AUTH_STATE.CREATING);
    setErrorMessage('');
    clearAuthorizationSession();
    try {
      const response = await seedanceAssetsApi.createAuthorizationSession();
      const session = response.data?.data;
      if (!session?.byted_token || !session?.h5_link) {
        throw new Error('invalid authorization session');
      }
      sessionStorage.setItem(AUTHORIZATION_TOKEN_KEY, session.byted_token);
      setH5Link(session.h5_link);
      setExpiresAt(session.expires_at || 0);
      pollStartedAtRef.current = Date.now();
      setAuthorizationState(AUTH_STATE.WAITING);
    } catch (error) {
      if (responseErrorCode(error) === 'asset_library_not_configured') {
        setAuthorizationState(AUTH_STATE.UNCONFIGURED);
        return;
      }
      setErrorMessage(t('授权链接创建失败，请稍后重试。'));
      setAuthorizationState(AUTH_STATE.FAILED);
    }
  }, [clearAuthorizationSession, consented, t]);

  useEffect(() => {
    if (authorizationState !== AUTH_STATE.WAITING) return undefined;
    const token = sessionStorage.getItem(AUTHORIZATION_TOKEN_KEY);
    if (!token) return undefined;
    if (!pollStartedAtRef.current) pollStartedAtRef.current = Date.now();

    const poll = async () => {
      const elapsedMs = Date.now() - pollStartedAtRef.current;
      if (!shouldContinueAuthorizationPolling(AUTH_STATE.WAITING, elapsedMs)) {
        clearAuthorizationSession();
        setErrorMessage(t('授权链接已过期，请重新生成。'));
        setAuthorizationState(AUTH_STATE.EXPIRED);
        return;
      }
      if (pollInFlightRef.current) return;
      pollInFlightRef.current = true;
      try {
        const response = await seedanceAssetsApi.authorizationResult(token);
        const result = response.data?.data;
        if (result?.authorized) {
          clearAuthorizationSession();
          setAuthorizationState(AUTH_STATE.AUTHORIZED);
        } else {
          setErrorMessage('');
        }
      } catch (error) {
        const nextState = authorizationStateFromPollError(
          error.response?.status,
          responseErrorCode(error),
        );
        if (nextState === AUTH_STATE.EXPIRED) {
          clearAuthorizationSession();
          setErrorMessage(t('授权链接已过期，请重新生成。'));
          setAuthorizationState(AUTH_STATE.EXPIRED);
        } else {
          setErrorMessage(t('授权查询失败，请重试。'));
          setAuthorizationState(AUTH_STATE.WAITING);
        }
      } finally {
        pollInFlightRef.current = false;
      }
    };

    poll();
    const timer = window.setInterval(poll, 3000);
    return () => window.clearInterval(timer);
  }, [authorizationState, clearAuthorizationSession, t]);

  const restartAuthorization = () => {
    clearAuthorizationSession();
    setErrorMessage('');
    if (consented) {
      startAuthorization();
    } else {
      setAuthorizationState(AUTH_STATE.UNAUTHORIZED);
    }
  };

  const requestReauthorization = () => {
    clearAuthorizationSession();
    setConsented(false);
    setErrorMessage('');
    setAuthorizationState(AUTH_STATE.UNAUTHORIZED);
  };

  return (
    <main className='mx-auto mt-[60px] min-h-screen w-full max-w-7xl px-3 pb-10 lg:min-h-0'>
      <Card className='mb-4'>
        <Title heading={2}>{t('真人素材库')}</Title>
        <Paragraph type='tertiary'>
          {t(
            '完成本人授权后，可登记与本人一致的图片或视频，并复制 asset:// 引用用于 Seedance 视频生成。',
          )}
        </Paragraph>
      </Card>

      {authorizationState === AUTH_STATE.CHECKING ? (
        <Card>
          <div className='flex min-h-48 items-center justify-center'>
            <Spin size='large' tip={t('正在检查授权状态')} />
          </div>
        </Card>
      ) : authorizationState === AUTH_STATE.AUTHORIZED ? (
        <AssetLibrary onReauthorize={requestReauthorization} />
      ) : (
        <AuthorizationCard
          state={authorizationState}
          consented={consented}
          onConsentChange={setConsented}
          h5Link={h5Link}
          expiresAt={expiresAt}
          errorMessage={errorMessage}
          onStart={startAuthorization}
          onRetry={restartAuthorization}
        />
      )}

      <Banner
        className='mt-4'
        type='warning'
        description={t(
          '本功能不提供绕过本人活体核验的方式；如无带摄像头的可用设备，将无法完成授权。',
        )}
      />
    </main>
  );
};

export default SeedanceAssets;
