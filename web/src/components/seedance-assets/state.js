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
  return state === AUTH_STATE.WAITING && elapsedMs < 300000;
}

export function authorizationStateFromPollError(status, errorCode) {
  if (status === 410 || errorCode === 'authorization_expired') {
    return AUTH_STATE.EXPIRED;
  }
  return AUTH_STATE.WAITING;
}
