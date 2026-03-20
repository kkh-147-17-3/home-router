import { useState, useCallback } from 'react'
import { Table, Tag, Input, Switch, Space, Button } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { usePolling } from '../../hooks/usePolling'
import { getDNSQueryLog } from '../../api/client'
import type { QueryEntry } from '../../api/types'

export default function QueryLogTable() {
  const [blockedOnly, setBlockedOnly] = useState(false)
  const [domain, setDomain] = useState('')

  const fetcher = useCallback(
    () => getDNSQueryLog({ limit: 200, blocked: blockedOnly || undefined, domain: domain || undefined }),
    [blockedOnly, domain],
  )
  const { data: entries, refresh } = usePolling(fetcher, 5000)

  const columns = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'time',
      width: 180,
      render: (v: string) => new Date(v).toLocaleTimeString(),
    },
    { title: 'Client', dataIndex: 'clientIp', key: 'client', width: 140 },
    { title: 'Domain', dataIndex: 'domain', key: 'domain', ellipsis: true },
    { title: 'Type', dataIndex: 'queryType', key: 'type', width: 60 },
    {
      title: 'Status',
      key: 'status',
      width: 100,
      render: (_: unknown, r: QueryEntry) => {
        if (r.blocked) return <Tag color="red">Blocked</Tag>
        if (r.cached) return <Tag color="blue">Cached</Tag>
        return <Tag color="green">Allowed</Tag>
      },
    },
    {
      title: 'Time (ms)',
      dataIndex: 'responseTimeMs',
      key: 'rt',
      width: 90,
      render: (v: number) => v.toFixed(1),
    },
  ]

  return (
    <div>
      <Space className="mb-4" wrap>
        <Input.Search placeholder="Search domain" allowClear onSearch={setDomain} style={{ width: 250 }} />
        <span>Blocked only: <Switch checked={blockedOnly} onChange={setBlockedOnly} /></span>
        <Button icon={<ReloadOutlined />} onClick={refresh}>Refresh</Button>
      </Space>
      <Table
        dataSource={entries || []}
        columns={columns}
        rowKey={(_, i) => String(i)}
        size="small"
        pagination={{ pageSize: 50, showSizeChanger: true }}
      />
    </div>
  )
}
