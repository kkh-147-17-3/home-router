import { useCallback, useState } from 'react'
import { Card, Form, Input, Select, Switch, Button, Space, Tag, Descriptions, message } from 'antd'
import { ReloadOutlined, CloudSyncOutlined } from '@ant-design/icons'
import { usePolling } from '../hooks/usePolling'
import { getDDNSStatus, updateDDNS, updateDDNSConfig } from '../api/client'
import type { DDNSStatus, DDNSConfigRequest } from '../api/types'

export default function DDNSPage() {
  const fetcher = useCallback(() => getDDNSStatus(), [])
  const { data: status, refresh } = usePolling(fetcher, 10000)
  const [form] = Form.useForm()
  const [editing, setEditing] = useState(false)
  const [provider, setProvider] = useState('')

  const handleManualUpdate = async () => {
    try {
      await updateDDNS()
      message.success('DDNS update triggered')
      refresh()
    } catch {
      message.error('Failed to trigger DDNS update')
    }
  }

  const handleSaveConfig = async (values: DDNSConfigRequest) => {
    try {
      await updateDDNSConfig(values)
      message.success('DDNS configuration saved')
      setEditing(false)
      refresh()
    } catch {
      message.error('Failed to save DDNS configuration')
    }
  }

  const startEditing = () => {
    if (status) {
      form.setFieldsValue({
        enabled: status.enabled,
        provider: status.provider || 'cloudflare',
        domain: status.domain,
        token: '',
        zoneId: '',
        recordId: '',
        proxied: false,
        updateUrl: '',
      })
      setProvider(status.provider || 'cloudflare')
    }
    setEditing(true)
  }

  return (
    <div className="space-y-4">
      <Card title="DDNS Status" extra={
        <Space>
          <Button icon={<CloudSyncOutlined />} onClick={handleManualUpdate}
            disabled={!status?.enabled}>
            Manual Update
          </Button>
          <Button icon={<ReloadOutlined />} onClick={refresh}>Refresh</Button>
        </Space>
      }>
        {status && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="Status">
              <Tag color={status.enabled ? 'green' : 'default'}>
                {status.enabled ? 'Enabled' : 'Disabled'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Provider">{status.provider || '-'}</Descriptions.Item>
            <Descriptions.Item label="Domain">{status.domain || '-'}</Descriptions.Item>
            <Descriptions.Item label="Current IP">{status.lastIp || '-'}</Descriptions.Item>
            <Descriptions.Item label="Last Update">
              {status.lastUpdate && status.lastUpdate !== '0001-01-01T00:00:00Z'
                ? new Date(status.lastUpdate).toLocaleString() : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Last Error">
              {status.lastError
                ? <Tag color="red">{status.lastError}</Tag>
                : <Tag color="green">None</Tag>}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Card>

      <Card title="DDNS Configuration" extra={
        !editing && <Button type="primary" onClick={startEditing}>Edit</Button>
      }>
        {editing ? (
          <Form form={form} layout="vertical" onFinish={handleSaveConfig}
            initialValues={{ enabled: false, provider: 'cloudflare', proxied: false }}>
            <Form.Item name="enabled" label="Enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="provider" label="Provider" rules={[{ required: true }]}>
              <Select onChange={(v) => setProvider(v)}>
                <Select.Option value="cloudflare">Cloudflare</Select.Option>
                <Select.Option value="duckdns">DuckDNS</Select.Option>
                <Select.Option value="custom">Custom</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item name="domain" label="Domain" rules={[{ required: true }]}>
              <Input placeholder="example.com" />
            </Form.Item>
            <Form.Item name="token" label="API Token">
              <Input.Password placeholder="Leave empty to keep existing" />
            </Form.Item>
            {provider === 'cloudflare' && (
              <>
                <Form.Item name="zoneId" label="Zone ID">
                  <Input />
                </Form.Item>
                <Form.Item name="recordId" label="Record ID">
                  <Input />
                </Form.Item>
                <Form.Item name="proxied" label="Proxied" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </>
            )}
            {provider === 'custom' && (
              <Form.Item name="updateUrl" label="Update URL"
                help="Use {{ip}}, {{domain}}, {{token}} as placeholders">
                <Input placeholder="https://provider.example/update?ip={{ip}}&domain={{domain}}" />
              </Form.Item>
            )}
            <Space>
              <Button type="primary" htmlType="submit">Save</Button>
              <Button onClick={() => setEditing(false)}>Cancel</Button>
            </Space>
          </Form>
        ) : (
          <p className="text-gray-500">Click Edit to configure DDNS settings.</p>
        )}
      </Card>
    </div>
  )
}
