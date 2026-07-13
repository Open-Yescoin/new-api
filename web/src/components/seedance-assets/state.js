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
export const AUTH_STATE = Object.freeze({
  CHECKING: 'checking',
  UNCONFIGURED: 'unconfigured',
  UNAUTHORIZED: 'unauthorized',
  CREATING: 'creating',
  WAITING: 'waiting',
  AUTHORIZED: 'authorized',
  EXPIRED: 'expired',
  FAILED: 'failed',
});

export const DEFAULT_AUTHORIZATION_DURATION_DAYS = 30;

export const AUTHORIZATION_DURATION_OPTIONS = Object.freeze([
  { value: 30, labelKey: '30 天授权' },
  { value: 60, labelKey: '60 天授权' },
  { value: 180, labelKey: '半年授权' },
  { value: 365, labelKey: '一年授权' },
  { value: null, labelKey: '长期有效（演员可随时撤回）' },
]);

export function authorizationExpiryWarningDays(expiresAt, nowSeconds) {
  if (!expiresAt || expiresAt <= nowSeconds) return null;
  const remaining = expiresAt - nowSeconds;
  if (remaining <= 86400) return 1;
  if (remaining <= 3 * 86400) return 3;
  if (remaining <= 7 * 86400) return 7;
  return null;
}

export function actorAuthorizationState(actor, nowSeconds) {
  const status = String(actor?.status || '').toLowerCase();
  if (status === 'revoked') return 'revoked';
  if (
    actor?.authorization_expires_at &&
    actor.authorization_expires_at <= nowSeconds
  ) {
    return 'expired';
  }
  if (status === 'expired') return 'expired';
  if (
    status === 'active' &&
    authorizationExpiryWarningDays(actor.authorization_expires_at, nowSeconds)
  ) {
    return 'expiring';
  }
  if (status === 'active') return 'active';
  if (status === 'legacyreview') return 'legacy_review';
  return 'pending';
}

const AUTHORIZATION_POLL_TIMEOUT_MS = 15 * 60 * 1000;

export function authorizationStateFromStatus(status) {
  if (!status) {
    return AUTH_STATE.CHECKING;
  }
  if (!status.configured) {
    return AUTH_STATE.UNCONFIGURED;
  }
  if (status.authorized) {
    return AUTH_STATE.AUTHORIZED;
  }
  if (status.pending) {
    return AUTH_STATE.WAITING;
  }
  return AUTH_STATE.UNAUTHORIZED;
}

export function shouldContinueAuthorizationPolling(state, elapsedMs) {
  return (
    state === AUTH_STATE.WAITING && elapsedMs < AUTHORIZATION_POLL_TIMEOUT_MS
  );
}

export function authorizationStateFromPollError(status, errorCode) {
  if (status === 410 || errorCode === 'authorization_expired') {
    return AUTH_STATE.EXPIRED;
  }
  return AUTH_STATE.WAITING;
}
