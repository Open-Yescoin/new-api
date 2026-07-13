import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  CheckCircle2,
  CircleX,
  Clock3,
  Copy,
  Pencil,
  RefreshCw,
  Trash2,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { copy, showError, showSuccess } from '../../helpers';
import { seedanceAssetsApi } from '../../services/seedanceAssets';

const { Paragraph, Text } = Typography;
const normalizedStatus = (asset) =>
  String(asset.Status || asset.UpstreamStatus || '').toLowerCase();

const AssetLibrary = ({ onReauthorize }) => {
  const { t } = useTranslation();
  const [assets, setAssets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [url, setUrl] = useState('');
  const [assetType, setAssetType] = useState('Image');
  const [renameAsset, setRenameAsset] = useState(null);
  const [renameValue, setRenameValue] = useState('');
  const [renaming, setRenaming] = useState(false);

  const loadAssets = useCallback(
    async ({ silent = false } = {}) => {
      if (!silent) setLoading(true);
      try {
        const response = await seedanceAssetsApi.list({
          page_number: 1,
          page_size: 100,
        });
        setAssets(response.data?.data?.Items || []);
      } catch (_error) {
        if (!silent) showError(t('素材列表加载失败，请稍后重试。'));
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    loadAssets();
  }, [loadAssets]);

  const hasProcessingAssets = useMemo(
    () => assets.some((asset) => normalizedStatus(asset) === 'processing'),
    [assets],
  );

  useEffect(() => {
    if (!hasProcessingAssets) return undefined;
    const refreshVisibleAssets = () => {
      if (document.visibilityState === 'visible') {
        loadAssets({ silent: true });
      }
    };
    const timer = window.setInterval(refreshVisibleAssets, 10000);
    document.addEventListener('visibilitychange', refreshVisibleAssets);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', refreshVisibleAssets);
    };
  }, [hasProcessingAssets, loadAssets]);

  const createAsset = async () => {
    let parsed;
    try {
      parsed = new URL(url.trim());
    } catch (_error) {
      showError(t('请输入有效的 HTTP(S) 公网素材地址。'));
      return;
    }
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      showError(t('请输入有效的 HTTP(S) 公网素材地址。'));
      return;
    }
    setCreating(true);
    try {
      await seedanceAssetsApi.create({
        url: parsed.toString(),
        asset_type: assetType,
      });
      setUrl('');
      showSuccess(t('素材已提交，正在处理。'));
      await loadAssets({ silent: true });
    } catch (error) {
      if (error.response?.status === 409) {
        showError(t('请先完成本人活体核验与肖像授权。'));
      } else {
        showError(t('素材提交失败，请检查地址和素材内容。'));
      }
    } finally {
      setCreating(false);
    }
  };

  const copyReference = async (asset) => {
    if (normalizedStatus(asset) !== 'active') return;
    if (await copy(`asset://${asset.Id}`)) {
      showSuccess(t('素材引用已复制'));
    } else {
      showError(t('复制失败，请手动复制素材引用。'));
    }
  };

  const openRename = (asset) => {
    setRenameAsset(asset);
    setRenameValue(asset.Name || '');
  };

  const submitRename = async () => {
    const name = renameValue.trim();
    if (!renameAsset || !name) {
      showError(t('请输入素材名称。'));
      return;
    }
    setRenaming(true);
    try {
      await seedanceAssetsApi.update(renameAsset.Id, name);
      setRenameAsset(null);
      showSuccess(t('素材名称已更新'));
      await loadAssets({ silent: true });
    } catch (_error) {
      showError(t('素材重命名失败，请稍后重试。'));
    } finally {
      setRenaming(false);
    }
  };

  const confirmDelete = (asset) => {
    Modal.confirm({
      title: t('确认删除素材？'),
      content: t('删除后无法恢复，并会使对应的 asset:// 引用失效。'),
      okType: 'danger',
      okText: t('删除'),
      cancelText: t('取消'),
      onOk: async () => {
        try {
          await seedanceAssetsApi.remove(asset.Id);
          showSuccess(t('素材已删除'));
          await loadAssets({ silent: true });
        } catch (error) {
          showError(t('素材删除失败，请稍后重试。'));
          throw error;
        }
      },
    });
  };

  const renderStatus = (asset) => {
    const status = normalizedStatus(asset);
    if (status === 'active') {
      return (
        <Tag color='green' prefixIcon={<CheckCircle2 size={14} />}>
          {t('可用')}
        </Tag>
      );
    }
    if (status === 'failed') {
      return (
        <div>
          <Tag color='red' prefixIcon={<CircleX size={14} />}>
            {t('处理失败')}
          </Tag>
          {asset.Error?.Message && (
            <Paragraph type='tertiary' size='small' className='mt-1'>
              {asset.Error.Message}
            </Paragraph>
          )}
        </div>
      );
    }
    return (
      <Tag color='blue' prefixIcon={<Clock3 size={14} />}>
        {t('处理中')}
      </Tag>
    );
  };

  const columns = [
    {
      title: t('名称'),
      dataIndex: 'Name',
      render: (name, asset) => name || asset.Id,
    },
    { title: t('类型'), dataIndex: 'AssetType' },
    { title: t('状态'), render: (_value, asset) => renderStatus(asset) },
    {
      title: t('创建时间'),
      dataIndex: 'CreateTime',
      render: (value) => value || '—',
    },
    {
      title: t('操作'),
      render: (_value, asset) => (
        <Space wrap>
          {normalizedStatus(asset) === 'active' && (
            <Button
              size='small'
              icon={<Copy size={14} />}
              onClick={() => copyReference(asset)}
            >
              {t('复制引用')}
            </Button>
          )}
          <Button
            size='small'
            icon={<Pencil size={14} />}
            onClick={() => openRename(asset)}
          >
            {t('重命名')}
          </Button>
          <Button
            size='small'
            type='danger'
            icon={<Trash2 size={14} />}
            onClick={() => confirmDelete(asset)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Space vertical spacing='medium' className='w-full'>
      <Card title={t('登记真人素材')}>
        <Banner
          type='info'
          description={t(
            '仅支持公网可访问的 HTTP(S) 图片或视频地址；素材需与已授权本人一致。',
          )}
        />
        <div className='mt-4 grid gap-3 md:grid-cols-[1fr_140px_auto]'>
          <div>
            <Text strong>{t('素材地址')}</Text>
            <Input
              className='mt-2'
              value={url}
              onChange={setUrl}
              placeholder='https://example.com/person.jpg'
              aria-label={t('素材地址')}
            />
          </div>
          <div>
            <Text strong>{t('素材类型')}</Text>
            <Select
              className='mt-2 w-full'
              value={assetType}
              onChange={setAssetType}
              aria-label={t('素材类型')}
              optionList={[
                { label: t('图片'), value: 'Image' },
                { label: t('视频'), value: 'Video' },
              ]}
            />
          </div>
          <Button
            className='md:self-end'
            type='primary'
            theme='solid'
            loading={creating}
            onClick={createAsset}
          >
            {t('登记素材')}
          </Button>
        </div>
      </Card>

      <Card
        title={t('我的真人素材')}
        headerExtraContent={
          <Space wrap>
            <Button icon={<RefreshCw size={15} />} onClick={() => loadAssets()}>
              {t('刷新')}
            </Button>
            <Button type='warning' onClick={onReauthorize}>
              {t('重新授权')}
            </Button>
          </Space>
        }
      >
        <Banner
          type='warning'
          description={t('重新授权会替换当前真人绑定，原素材不会自动迁移。')}
          className='mb-4'
        />
        <Table
          columns={columns}
          dataSource={assets}
          rowKey='Id'
          loading={loading}
          pagination={{ pageSize: 10 }}
          scroll={{ x: 900 }}
          empty={<Text type='tertiary'>{t('暂无真人素材')}</Text>}
        />
      </Card>

      <Modal
        visible={Boolean(renameAsset)}
        title={t('重命名素材')}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={renaming}
        onOk={submitRename}
        onCancel={() => setRenameAsset(null)}
      >
        <Text strong>{t('素材名称')}</Text>
        <Input
          className='mt-2'
          value={renameValue}
          onChange={setRenameValue}
          aria-label={t('素材名称')}
        />
      </Modal>
    </Space>
  );
};

export default AssetLibrary;
