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
import { describe, expect, test } from 'bun:test';
import {
  AUTH_STATE,
  authorizationStateFromStatus,
  authorizationStateFromPollError,
  actorAuthorizationState,
  authorizationExpiryWarningDays,
  AUTHORIZATION_DURATION_OPTIONS,
  DEFAULT_AUTHORIZATION_DURATION_DAYS,
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

  test('authorization periods are fixed and default to thirty days', () => {
    expect(
      AUTHORIZATION_DURATION_OPTIONS.map((option) => option.value),
    ).toEqual([30, 60, 180, 365, null]);
    expect(DEFAULT_AUTHORIZATION_DURATION_DAYS).toBe(30);
  });

  test('actor expiry is immediate and warning buckets are deterministic', () => {
    expect(
      actorAuthorizationState(
        { status: 'Active', authorization_expires_at: 100 },
        100,
      ),
    ).toBe('expired');
    expect(
      actorAuthorizationState(
        { status: 'Revoked', authorization_expires_at: 200 },
        100,
      ),
    ).toBe('revoked');
    expect(
      actorAuthorizationState(
        { status: 'Active', authorization_expires_at: 100 + 7 * 86400 },
        100,
      ),
    ).toBe('expiring');
    expect(authorizationExpiryWarningDays(100 + 86400, 100)).toBe(1);
    expect(authorizationExpiryWarningDays(100 + 3 * 86400, 100)).toBe(3);
    expect(authorizationExpiryWarningDays(100 + 7 * 86400, 100)).toBe(7);
    expect(authorizationExpiryWarningDays(100 + 8 * 86400, 100)).toBeNull();
  });
});
