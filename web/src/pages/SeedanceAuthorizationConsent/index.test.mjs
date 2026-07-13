import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

describe('Seedance actor mobile consent handoff', () => {
  test('keeps secrets out of server-visible URLs and requires consent before H5', () => {
    expect(source).toContain('window.location.hash');
    expect(source).toContain('window.history.replaceState');
    expect(source).toContain('publicInvitationDetails');
    expect(source).toContain('publicAcceptConsent');
    expect(source).toContain('sessionStorage.setItem');
    expect(source).toContain('window.location.assign(h5Link)');
    expect(source.indexOf('publicAcceptConsent')).toBeLessThan(
      source.indexOf('window.location.assign(h5Link)'),
    );
    expect(source).not.toContain('?token=');
  });
});
