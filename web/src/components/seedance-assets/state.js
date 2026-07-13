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
