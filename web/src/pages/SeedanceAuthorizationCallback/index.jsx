import React from 'react';
import { Card, Space, Typography } from '@douyinfe/semi-ui';
import { CheckCircle2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

const { Paragraph, Title } = Typography;

const SeedanceAuthorizationCallback = () => {
  const { t } = useTranslation();

  return (
    <main className='mx-auto flex min-h-screen max-w-xl items-center px-4 py-10'>
      <Card className='w-full text-center'>
        <Space vertical spacing='medium' align='center'>
          <CheckCircle2
            size={48}
            color='var(--semi-color-success)'
            aria-hidden='true'
          />
          <Title heading={2}>{t('授权完成，请返回原设备。')}</Title>
          <Paragraph>
            {t(
              '手机端无需登录 Token123。原设备会自动查询授权结果并进入下一步。',
            )}
          </Paragraph>
          <Paragraph type='tertiary'>
            {t('请关闭本页面并返回原设备。')}
          </Paragraph>
        </Space>
      </Card>
    </main>
  );
};

export default SeedanceAuthorizationCallback;
