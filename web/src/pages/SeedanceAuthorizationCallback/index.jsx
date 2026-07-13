import React from 'react';
import { Button, Card, Space, Typography } from '@douyinfe/semi-ui';
import { CheckCircle2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';

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
              '原设备会自动查询授权结果；如果您在本手机开始授权，也可以返回真人素材库。',
            )}
          </Paragraph>
          <Link to='/console/seedance-assets'>
            <Button type='primary' theme='solid'>
              {t('返回真人素材库')}
            </Button>
          </Link>
        </Space>
      </Card>
    </main>
  );
};

export default SeedanceAuthorizationCallback;
