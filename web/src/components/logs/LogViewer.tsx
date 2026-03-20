import { useState, useCallback } from 'react'
import { Table, Select, Input, Button, Space, Tag } from 'antd'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { usePolling } from '../../hooks/usePolling'
import { getSystemLogs } from '../../api/client'
import type { LogEntry } from '../../api/types'

const units = [
  { value: 'home-router', label: 'home-router' },
  { value: 'systemd-networkd', label: 'systemd-networkd' },
  { value: 'systemd-resolved', label: 'systemd-resolved' },
]

const priorities = [
  { value: '', label: 'All' },
  { value: 'err', label: 'Error' },
  { value: 'warning', label: 'Warning' },
  { value: 'info', label: 'Info' },
  { value: 'debug', label: 'Debug' },
]

const priorityColor: Record<string, string> = {
  emerg: 'red', alert: 'red', crit: 'red', err: 'red',
  warning: 'orange', notice: 'blue', info: 'blue', debug: 'default',
}

export default function LogViewer() {
  const [unit, setUnit] = useState('home-router')
  const [lines, setLines] = useState('100')
  const [priority, setPriority] = useState('')
  const [grep, setGrep] = useState('')

  const fetcher = useCallback(
    () => getSystemLogs({ unit, lines, priority: priority || undefined, grep: grep || undefined }),
    [unit, lines, priority, grep],
  )
  const { data: logs, refresh } = usePolling(fetcher, 30000)

  const columns = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'time',
      width: 200,
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: 'Level',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (v: string) => <Tag color={priorityColor[v] || 'default'}>{v}</Tag>,
    },
    { title: 'Message', dataIndex: 'message', key: 'message', ellipsis: true },
  ]

  return (
    <div>
      <Space className="mb-4" wrap>
        <Select value={unit} onChange={setUnit} options={units} style={{ width: 180 }} />
        <Select value={priority} onChange={setPriority} options={priorities} style={{ width: 120 }} />
        <Select value={lines} onChange={setLines} style={{ width: 100 }}
          options={[
            { value: '50', label: '50 lines' },
            { value: '100', label: '100 lines' },
            { value: '500', label: '500 lines' },
          ]}
        />
        <Input
          prefix={<SearchOutlined />}
          placeholder="Filter text"
          value={grep}
          onChange={(e) => setGrep(e.target.value)}
          onPressEnter={refresh}
          style={{ width: 200 }}
          allowClear
        />
        <Button icon={<ReloadOutlined />} onClick={refresh}>Refresh</Button>
      </Space>
      <Table
        dataSource={logs || []}
        columns={columns}
        rowKey={(_, i) => String(i)}
        size="small"
        pagination={{ pageSize: 50, showSizeChanger: true }}
      />
    </div>
  )
}
