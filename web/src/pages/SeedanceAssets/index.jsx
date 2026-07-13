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
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Banner, Card, Space, Spin, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import ActorManager from '../../components/seedance-assets/ActorManager';
import AuthorizationCard from '../../components/seedance-assets/AuthorizationCard';
import AssetLibrary from '../../components/seedance-assets/AssetLibrary';
import {
  AUTH_STATE,
  actorAuthorizationState,
  authorizationExpiryWarningDays,
  authorizationStateFromPollError,
  shouldContinueAuthorizationPolling,
} from '../../components/seedance-assets/state';
import { showError, showSuccess } from '../../helpers';
import { seedanceAssetsApi } from '../../services/seedanceAssets';

const { Paragraph, Title } = Typography;
const responseErrorCode = (error) => error.response?.data?.error?.code;
const authorizationStorageKey = (actorId) =>
  `seedance_actor_authorization_${actorId}`;

const readStoredSession = (actorId) => {
  if (!actorId) return null;
  try {
    return JSON.parse(
      sessionStorage.getItem(authorizationStorageKey(actorId)) || 'null',
    );
  } catch (_error) {
    sessionStorage.removeItem(authorizationStorageKey(actorId));
    return null;
  }
};

const SeedanceAssets = () => {
  const { t } = useTranslation();
  const [actors, setActors] = useState([]);
  const [selectedActorId, setSelectedActorId] = useState(null);
  const [actorsLoading, setActorsLoading] = useState(true);
  const [authorizationState, setAuthorizationState] = useState(
    AUTH_STATE.CHECKING,
  );
  const [handoffUrl, setHandoffUrl] = useState('');
  const [expiresAt, setExpiresAt] = useState(0);
  const [errorMessage, setErrorMessage] = useState('');
  const pollStartedAtRef = useRef(0);
  const pollInFlightRef = useRef(false);

  const selectedActor = useMemo(
    () => actors.find((actor) => actor.id === selectedActorId) || null,
    [actors, selectedActorId],
  );

  const loadActors = useCallback(
    async (preferredActorId = null) => {
      setActorsLoading(true);
      try {
        const response = await seedanceAssetsApi.listActors();
        const nextActors = response.data?.data || [];
        setActors(nextActors);
        setSelectedActorId((current) => {
          const preferred = preferredActorId || current;
          if (nextActors.some((actor) => actor.id === preferred)) {
            return preferred;
          }
          return nextActors[0]?.id || null;
        });
      } catch (_error) {
        showError(t('演员列表加载失败，请稍后重试。'));
      } finally {
        setActorsLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    loadActors();
  }, [loadActors]);

  const clearAuthorizationSession = useCallback((actorId) => {
    if (actorId) {
      sessionStorage.removeItem(authorizationStorageKey(actorId));
    }
    setHandoffUrl('');
    setExpiresAt(0);
    pollStartedAtRef.current = 0;
  }, []);

  const loadAuthorizationStatus = useCallback(async () => {
    if (!selectedActorId) {
      setAuthorizationState(AUTH_STATE.UNAUTHORIZED);
      return;
    }
    setAuthorizationState(AUTH_STATE.CHECKING);
    try {
      const response =
        await seedanceAssetsApi.actorAuthorizationStatus(selectedActorId);
      const status = response.data?.data;
      const actor = status?.actor;
      if (actor) {
        setActors((current) =>
          current.map((item) => (item.id === actor.id ? actor : item)),
        );
      }
      if (!status?.configured) {
        setAuthorizationState(AUTH_STATE.UNCONFIGURED);
        return;
      }
      const stored = readStoredSession(selectedActorId);
      if (status.pending && stored?.bytedToken) {
        setHandoffUrl(stored.handoffUrl || '');
        setExpiresAt(stored.expiresAt || 0);
        pollStartedAtRef.current = Date.now();
        setAuthorizationState(AUTH_STATE.WAITING);
        return;
      }
      if (stored) clearAuthorizationSession(selectedActorId);
      const actorState = actorAuthorizationState(
        actor,
        Math.floor(Date.now() / 1000),
      );
      setAuthorizationState(
        actorState === 'active' || actorState === 'expiring'
          ? AUTH_STATE.AUTHORIZED
          : AUTH_STATE.UNAUTHORIZED,
      );
    } catch (_error) {
      setErrorMessage(t('授权状态加载失败，请稍后重试。'));
      setAuthorizationState(AUTH_STATE.FAILED);
    }
  }, [clearAuthorizationSession, selectedActorId, t]);

  useEffect(() => {
    setErrorMessage('');
    setHandoffUrl('');
    setExpiresAt(0);
    loadAuthorizationStatus();
  }, [loadAuthorizationStatus]);

  const startAuthorization = useCallback(async () => {
    if (!selectedActorId) return;
    setAuthorizationState(AUTH_STATE.CREATING);
    setErrorMessage('');
    clearAuthorizationSession(selectedActorId);
    try {
      const response =
        await seedanceAssetsApi.createActorAuthorizationSession(
          selectedActorId,
        );
      const session = response.data?.data;
      if (
        !session?.byted_token ||
        !session?.h5_link ||
        !session?.invitation_token ||
        !session?.revocation_token
      ) {
        throw new Error('invalid actor authorization session');
      }
      const fragment = new URLSearchParams({
        invitation: session.invitation_token,
        h5: session.h5_link,
        revoke: session.revocation_token,
      });
      const nextHandoffUrl = `${window.location.origin}/seedance/authorization/consent#${fragment}`;
      const stored = {
        bytedToken: session.byted_token,
        handoffUrl: nextHandoffUrl,
        expiresAt: session.expires_at || 0,
      };
      sessionStorage.setItem(
        authorizationStorageKey(selectedActorId),
        JSON.stringify(stored),
      );
      setHandoffUrl(nextHandoffUrl);
      setExpiresAt(stored.expiresAt);
      pollStartedAtRef.current = Date.now();
      setAuthorizationState(AUTH_STATE.WAITING);
    } catch (error) {
      if (responseErrorCode(error) === 'asset_library_not_configured') {
        setAuthorizationState(AUTH_STATE.UNCONFIGURED);
        return;
      }
      setErrorMessage(t('授权邀请创建失败，请稍后重试。'));
      setAuthorizationState(AUTH_STATE.FAILED);
    }
  }, [clearAuthorizationSession, selectedActorId, t]);

  useEffect(() => {
    if (authorizationState !== AUTH_STATE.WAITING || !selectedActorId) {
      return undefined;
    }
    const stored = readStoredSession(selectedActorId);
    if (!stored?.bytedToken) return undefined;
    if (!pollStartedAtRef.current) pollStartedAtRef.current = Date.now();

    const poll = async () => {
      const elapsedMs = Date.now() - pollStartedAtRef.current;
      if (!shouldContinueAuthorizationPolling(AUTH_STATE.WAITING, elapsedMs)) {
        clearAuthorizationSession(selectedActorId);
        setErrorMessage(t('授权邀请已过期，请重新生成。'));
        setAuthorizationState(AUTH_STATE.EXPIRED);
        return;
      }
      if (pollInFlightRef.current) return;
      pollInFlightRef.current = true;
      try {
        const response = await seedanceAssetsApi.actorAuthorizationResult(
          selectedActorId,
          stored.bytedToken,
        );
        if (response.data?.data?.authorized) {
          clearAuthorizationSession(selectedActorId);
          setAuthorizationState(AUTH_STATE.AUTHORIZED);
          showSuccess(t('演员真人授权已完成'));
          await loadActors(selectedActorId);
        } else {
          setErrorMessage('');
        }
      } catch (error) {
        const nextState = authorizationStateFromPollError(
          error.response?.status,
          responseErrorCode(error),
        );
        if (nextState === AUTH_STATE.EXPIRED) {
          clearAuthorizationSession(selectedActorId);
          setErrorMessage(t('授权邀请已过期，请重新生成。'));
          setAuthorizationState(AUTH_STATE.EXPIRED);
        } else if (
          responseErrorCode(error) === 'actor_authorization_required'
        ) {
          setErrorMessage(t('等待演员在手机上确认授权条款。'));
        } else {
          setErrorMessage(t('授权查询失败，系统会继续重试。'));
        }
      } finally {
        pollInFlightRef.current = false;
      }
    };

    poll();
    const timer = window.setInterval(poll, 3000);
    return () => window.clearInterval(timer);
  }, [
    authorizationState,
    clearAuthorizationSession,
    loadActors,
    selectedActorId,
    t,
  ]);

  const createActor = async (payload) => {
    try {
      const response = await seedanceAssetsApi.createActor(payload);
      const actor = response.data?.data;
      showSuccess(t('演员档案已创建'));
      await loadActors(actor?.id || null);
    } catch (_error) {
      showError(t('演员档案创建失败，请检查名称和授权期限。'));
      throw _error;
    }
  };

  const renameActor = async (actorId, displayName) => {
    try {
      await seedanceAssetsApi.renameActor(actorId, displayName);
      showSuccess(t('演员名称已更新'));
      await loadActors(actorId);
    } catch (_error) {
      showError(t('演员名称更新失败。'));
      throw _error;
    }
  };

  const revokeActor = async (actorId) => {
    try {
      await seedanceAssetsApi.revokeActor(actorId);
      clearAuthorizationSession(actorId);
      showSuccess(t('演员授权已撤回并立即停用'));
      await loadActors(actorId);
      setAuthorizationState(AUTH_STATE.UNAUTHORIZED);
    } catch (_error) {
      showError(t('撤回授权失败，请稍后重试。'));
      throw _error;
    }
  };

  const actorState = actorAuthorizationState(
    selectedActor,
    Math.floor(Date.now() / 1000),
  );
  const showLibrary = selectedActor && actorState !== 'pending';
  const showAuthorization =
    selectedActor && authorizationState !== AUTH_STATE.AUTHORIZED;
  const expiryWarningDays = authorizationExpiryWarningDays(
    selectedActor?.authorization_expires_at,
    Math.floor(Date.now() / 1000),
  );

  return (
    <main className='mx-auto mt-[60px] min-h-screen w-full max-w-7xl px-3 pb-10 lg:min-h-0'>
      <Card className='mb-4'>
        <Title heading={2}>{t('真人素材库')}</Title>
        <Paragraph type='tertiary'>
          {t(
            '一个账号可以管理多位演员；每位演员独立授权、独立到期，并拥有独立真人素材组。',
          )}
        </Paragraph>
      </Card>

      <Space vertical spacing='medium' className='w-full'>
        <ActorManager
          actors={actors}
          selectedActorId={selectedActorId}
          loading={actorsLoading}
          onSelect={setSelectedActorId}
          onCreate={createActor}
          onRename={renameActor}
          onRevoke={revokeActor}
        />

        {expiryWarningDays && (
          <Banner
            type='warning'
            description={t('授权将在 {{days}} 天内到期，请安排演员重新确认。', {
              days: expiryWarningDays,
            })}
          />
        )}

        {selectedActorId && authorizationState === AUTH_STATE.CHECKING ? (
          <Card>
            <div className='flex min-h-40 items-center justify-center'>
              <Spin size='large' tip={t('正在检查演员授权状态')} />
            </div>
          </Card>
        ) : showAuthorization ? (
          <AuthorizationCard
            state={authorizationState}
            actor={selectedActor}
            handoffUrl={handoffUrl}
            expiresAt={expiresAt}
            errorMessage={errorMessage}
            onStart={startAuthorization}
            onRetry={startAuthorization}
          />
        ) : null}

        {showLibrary && (
          <AssetLibrary
            key={selectedActor.id}
            actor={selectedActor}
            onReauthorize={() => setAuthorizationState(AUTH_STATE.UNAUTHORIZED)}
          />
        )}

        <Banner
          type='warning'
          description={t(
            '每位演员必须本人完成授权；到期或撤回会立即停止新建素材和视频调用，其他演员不受影响。',
          )}
        />
      </Space>
    </main>
  );
};

export default SeedanceAssets;
