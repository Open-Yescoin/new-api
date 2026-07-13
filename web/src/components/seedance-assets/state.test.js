import { describe, expect, test } from 'bun:test';
import {
  AUTH_STATE,
  authorizationStateFromStatus,
  authorizationStateFromPollError,
  shouldContinueAuthorizationPolling,
} from './state';

describe('Seedance authorization state', () => {
  test('configuration and authorization have deterministic priority', () => {
    expect(authorizationStateFromStatus({ configured: false })).toBe(
      AUTH_STATE.UNCONFIGURED,
    );
    expect(
      authorizationStateFromStatus({ configured: true, authorized: true }),
    ).toBe(AUTH_STATE.AUTHORIZED);
    expect(
      authorizationStateFromStatus({ configured: true, pending: true }),
    ).toBe(AUTH_STATE.WAITING);
    expect(authorizationStateFromStatus({ configured: true })).toBe(
      AUTH_STATE.UNAUTHORIZED,
    );
  });

  test('polling ends at a terminal state or fifteen minutes', () => {
    expect(shouldContinueAuthorizationPolling(AUTH_STATE.WAITING, 899999)).toBe(
      true,
    );
    expect(shouldContinueAuthorizationPolling(AUTH_STATE.WAITING, 900000)).toBe(
      false,
    );
    expect(
      shouldContinueAuthorizationPolling(AUTH_STATE.AUTHORIZED, 1000),
    ).toBe(false);
  });

  test('temporary polling failures preserve the current session', () => {
    expect(authorizationStateFromPollError(503, 'asset_upstream_error')).toBe(
      AUTH_STATE.WAITING,
    );
    expect(authorizationStateFromPollError(410, 'authorization_expired')).toBe(
      AUTH_STATE.EXPIRED,
    );
  });
});
