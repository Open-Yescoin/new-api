import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const componentPath = new URL(
  './components/settings/VolcAssetSetting.jsx',
  import.meta.url,
);
const settingPagePath = new URL('./pages/Setting/index.jsx', import.meta.url);

test('Seedance asset settings keep the secret write-only', async () => {
  const source = await readFile(componentPath, 'utf8');

  assert.match(source, /VolcAssetConfig/);
  assert.match(source, /secret_key_configured/);
  assert.match(source, /secret_key:\s*''/);
  assert.match(source, /mode=['"]password['"]/);
  assert.match(source, /group_type:\s*['"]LivenessFace['"]/);
});

test('root settings expose a dedicated Seedance asset tab', async () => {
  const source = await readFile(settingPagePath, 'utf8');

  assert.match(source, /import VolcAssetSetting/);
  assert.match(source, /content:\s*<VolcAssetSetting\s*\/>/);
  assert.match(source, /itemKey:\s*['"]seedance-assets['"]/);
});
