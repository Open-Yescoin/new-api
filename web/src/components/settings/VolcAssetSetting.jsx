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

import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Form,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;
const OPTION_KEY = 'VolcAssetConfig';
const DEFAULT_VALUES = {
  access_key: '',
  secret_key: '',
  secret_key_configured: false,
  region: 'ap-southeast-1',
  project_name: 'default',
  group_type: 'LivenessFace',
};

const VolcAssetSetting = () => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [secretConfigured, setSecretConfigured] = useState(false);

  const loadSettings = async () => {
    setLoading(true);
    try {
      const response = await API.get('/api/option/');
      const { success, message, data } = response.data;
      if (!success) {
        showError(message);
        return;
      }

      const option = data.find((item) => item.key === OPTION_KEY);
      const savedValues = option?.value ? JSON.parse(option.value) : {};
      const values = {
        ...DEFAULT_VALUES,
        ...savedValues,
        secret_key: '',
        group_type: 'LivenessFace',
      };
      setSecretConfigured(Boolean(savedValues.secret_key_configured));
      formApiRef.current?.setValues(values);
    } catch (error) {
      showError(error.message || t('加载 Seedance 素材库配置失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSettings();
  }, []);

  const saveSettings = async (values) => {
    setLoading(true);
    try {
      const secretKey = values.secret_key?.trim() || '';
      const config = {
        access_key: values.access_key.trim(),
        secret_key: secretKey,
        region: values.region.trim(),
        project_name: values.project_name.trim(),
        group_type: 'LivenessFace',
      };
      const response = await API.put('/api/option/', {
        key: OPTION_KEY,
        value: JSON.stringify(config),
      });
      const { success, message } = response.data;
      if (!success) {
        showError(message);
        return;
      }

      setSecretConfigured(secretConfigured || secretKey !== '');
      formApiRef.current?.setValue('secret_key', '');
      showSuccess(t('Seedance 素材库配置已更新'));
    } catch (error) {
      showError(error.message || t('更新 Seedance 素材库配置失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Spin spinning={loading} size='large'>
      <Card style={{ marginTop: '10px' }}>
        <Form
          initValues={DEFAULT_VALUES}
          getFormApi={(api) => (formApiRef.current = api)}
          onSubmit={saveSettings}
        >
          <Banner
            type='info'
            className='mb-4'
            description={t(
              '真人素材必须先完成 BytePlus H5 授权与活体校验，再注册为 LivenessFace 素材。普通上传不能直接用于包含真人脸部的 Seedance 2.0 生成。',
            )}
          />
          <Form.Input
            field='access_key'
            label={t('BytePlus Access Key')}
            rules={[
              { required: true, message: t('请输入 BytePlus Access Key') },
            ]}
          />
          <Form.Input
            field='secret_key'
            mode='password'
            label={t('BytePlus Secret Key')}
            extraText={
              secretConfigured
                ? t('密钥已配置；留空保存将保留现有密钥。')
                : t('尚未配置密钥。密钥保存后不会回显。')
            }
          />
          <Form.Input
            field='region'
            label={t('BytePlus 区域')}
            rules={[{ required: true, message: t('请输入 BytePlus 区域') }]}
          />
          <Form.Input
            field='project_name'
            label={t('BytePlus 项目名称')}
            rules={[{ required: true, message: t('请输入 BytePlus 项目名称') }]}
          />
          <Form.Input field='group_type' label={t('素材组类型')} disabled />
          <Text type='tertiary'>
            {t(
              '保存后，用户可通过受 Token 鉴权保护的 /doubao/open 素材接口完成授权、注册、查询与管理。',
            )}
          </Text>
          <div className='mt-4'>
            <Button htmlType='submit' type='primary' loading={loading}>
              {t('保存 Seedance 素材库配置')}
            </Button>
          </div>
        </Form>
      </Card>
    </Spin>
  );
};

export default VolcAssetSetting;
