import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

describe('Seedance mobile authorization callback', () => {
  test('keeps the phone on a public completion page', () => {
    expect(source).toContain('手机端无需登录 Token123');
    expect(source).toContain('请关闭本页面并返回原设备');
    expect(source).not.toContain("to='/console/seedance-assets'");
    expect(source).not.toContain("from 'react-router-dom'");
  });
});
