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
import React, { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Input,
  Modal,
  Select,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Pencil, Plus, ShieldOff } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { showError } from '../../helpers';
import {
  actorAuthorizationState,
  AUTHORIZATION_DURATION_OPTIONS,
  DEFAULT_AUTHORIZATION_DURATION_DAYS,
} from './state';

const { Text } = Typography;
const LONG_TERM_VALUE = 'long_term';

const actorStatusLabel = (actor, t) => {
  const status = actorAuthorizationState(actor, Math.floor(Date.now() / 1000));
  const labels = {
    active: [t('有效'), 'green'],
    expiring: [t('即将到期'), 'orange'],
    expired: [t('已过期'), 'red'],
    revoked: [t('已撤回'), 'red'],
    legacy_review: [t('需要重新确认'), 'orange'],
    pending: [t('待授权'), 'grey'],
  };
  return labels[status] || labels.pending;
};

const ActorManager = ({
  actors,
  selectedActorId,
  loading,
  onSelect,
  onCreate,
  onRename,
  onRevoke,
}) => {
  const { t } = useTranslation();
  const [createVisible, setCreateVisible] = useState(false);
  const [displayName, setDisplayName] = useState('');
  const [durationValue, setDurationValue] = useState(
    String(DEFAULT_AUTHORIZATION_DURATION_DAYS),
  );
  const [saving, setSaving] = useState(false);
  const [renameVisible, setRenameVisible] = useState(false);
  const [renameValue, setRenameValue] = useState('');

  const selectedActor = useMemo(
    () => actors.find((actor) => actor.id === selectedActorId) || null,
    [actors, selectedActorId],
  );

  const options = actors.map((actor) => {
    const [status] = actorStatusLabel(actor, t);
    return {
      value: actor.id,
      label: `${actor.display_name} · ${status}`,
    };
  });

  const submitCreate = async () => {
    const name = displayName.trim();
    if (!name) {
      showError(t('请输入演员名称。'));
      return;
    }
    setSaving(true);
    try {
      await onCreate({
        display_name: name,
        duration_days:
          durationValue === LONG_TERM_VALUE ? null : Number(durationValue),
      });
      setCreateVisible(false);
      setDisplayName('');
      setDurationValue(String(DEFAULT_AUTHORIZATION_DURATION_DAYS));
    } finally {
      setSaving(false);
    }
  };

  const submitRename = async () => {
    const name = renameValue.trim();
    if (!selectedActor || !name) return;
    setSaving(true);
    try {
      await onRename(selectedActor.id, name);
      setRenameVisible(false);
    } finally {
      setSaving(false);
    }
  };

  const confirmRevoke = () => {
    if (!selectedActor) return;
    Modal.confirm({
      title: t('确认撤回该演员授权？'),
      content: t('撤回后会立即停止新建素材和使用该演员素材生成视频。'),
      okType: 'danger',
      okText: t('立即撤回'),
      cancelText: t('取消'),
      onOk: () => onRevoke(selectedActor.id),
    });
  };

  const selectedStatus = selectedActor
    ? actorStatusLabel(selectedActor, t)
    : null;

  return (
    <Card
      title={t('演员档案')}
      headerExtraContent={
        <Button
          icon={<Plus size={15} />}
          onClick={() => setCreateVisible(true)}
        >
          {t('新增演员')}
        </Button>
      }
    >
      <Space vertical align='start' spacing='medium' className='w-full'>
        {actors.length ? (
          <div className='grid w-full gap-3 md:grid-cols-[minmax(220px,1fr)_auto]'>
            <Select
              className='w-full'
              value={selectedActorId}
              optionList={options}
              loading={loading}
              onChange={onSelect}
              aria-label={t('选择演员')}
            />
            <Space wrap>
              {selectedStatus && (
                <Tag color={selectedStatus[1]}>{selectedStatus[0]}</Tag>
              )}
              <Button
                icon={<Pencil size={14} />}
                disabled={!selectedActor}
                onClick={() => {
                  setRenameValue(selectedActor?.display_name || '');
                  setRenameVisible(true);
                }}
              >
                {t('重命名')}
              </Button>
              <Button
                type='danger'
                icon={<ShieldOff size={14} />}
                disabled={!selectedActor}
                onClick={confirmRevoke}
              >
                {t('撤回授权')}
              </Button>
            </Space>
          </div>
        ) : (
          <Text type='tertiary'>
            {t('尚未创建演员，请先新增演员并由本人完成授权。')}
          </Text>
        )}
      </Space>

      <Modal
        visible={createVisible}
        title={t('新增演员')}
        okText={t('创建')}
        cancelText={t('取消')}
        confirmLoading={saving}
        onOk={submitCreate}
        onCancel={() => setCreateVisible(false)}
      >
        <Space vertical align='start' spacing='medium' className='w-full'>
          <div className='w-full'>
            <Text strong>{t('演员名称')}</Text>
            <Input
              className='mt-2'
              value={displayName}
              onChange={setDisplayName}
              maxLength={128}
              aria-label={t('演员名称')}
            />
          </div>
          <div className='w-full'>
            <Text strong>{t('授权期限')}</Text>
            <Select
              className='mt-2 w-full'
              value={durationValue}
              onChange={setDurationValue}
              optionList={AUTHORIZATION_DURATION_OPTIONS.map((option) => ({
                value:
                  option.value === null
                    ? LONG_TERM_VALUE
                    : String(option.value),
                label: t(option.labelKey),
              }))}
            />
          </div>
          <Text type='tertiary'>
            {t('期限不能由公司单方面延长；到期后必须由演员重新确认。')}
          </Text>
        </Space>
      </Modal>

      <Modal
        visible={renameVisible}
        title={t('重命名演员')}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={saving}
        onOk={submitRename}
        onCancel={() => setRenameVisible(false)}
      >
        <Input
          value={renameValue}
          onChange={setRenameValue}
          maxLength={128}
          aria-label={t('演员名称')}
        />
      </Modal>
    </Card>
  );
};

export default ActorManager;
