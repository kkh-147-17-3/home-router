import { useState, useRef, useEffect } from 'react'
import { Table, Button, Space, Switch, Tag } from 'antd'
import { PauseOutlined, CaretRightOutlined, ClearOutlined } from '@ant-design/icons'
import { useSSE } from '../../hooks/useSSE'
import type { QueryEntry } from '../../api/types'

export default function LiveDNSStream() {
  const [enabled, setEnabled] = useState(true)
  const [blockedOnly, setBlockedOnly] = useState(false)
  const tableRef = useRef<HTMLDivElement>(null)

  const { data: entries, connected, clear } = useSSE<QueryEntry>(
    '/api/sse/dns-querylog',
    { enabled },
  )

  const filtered = blockedOnly ? entries.filter(e => e.blocked) : entries
  const display = filtered.slice().reverse() // newest first

  useEffect(() => {
    if (tableRef.current) {
      const tbody = tableRef.current.querySelector('.ant-table-body')
      if (tbody) tbody.scrollTop = 0
    }
  }, [entries])

  const columns = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'time',
      width: 100,
      render: (v: string) => new Date(v).toLocaleTimeString(),
    },
    { title: 'Client', dataIndex: 'client_ip', key: 'client', width: 130 },
    { title: 'Domain', dataIndex: 'domain', key: 'domain', ellipsis: true },
    { title: 'Type', dataIndex: 'query_type', key: 'type', width: 60 },
    {
      title: 'Status',
      key: 'status',
      width: 90,
      render: (_: unknown, r: QueryEntry) => {
        if (r.blocked) return <Tag color="red">Blocked</Tag>
        if (r.cached) return <Tag color="blue">Cached</Tag>
        return <Tag color="green">OK</Tag>
      },
    },
  ]

  return (
    <div ref={tableRef}>
      <Space className="mb-4" wrap>
        <Button
          icon={enabled ? <PauseOutlined /> : <CaretRightOutlined />}
          onClick={() => setEnabled(!enabled)}
        >
          {enabled ? 'Pause' : 'Resume'}
        </Button>
        <Button icon={<ClearOutlined />} onClick={clear}>Clear</Button>
        <span>Blocked only: <Switch checked={blockedOnly} onChange={setBlockedOnly} size="small" /></span>
        <Tag color={connected ? 'green' : 'red'}>{connected ? 'Connected' : 'Disconnected'}</Tag>
        <span className="text-gray-500 text-sm">{entries.length} entries</span>
      </Space>
      <Table
        dataSource={display}
        columns={columns}
        rowKey={(_, i) => String(i)}
        size="small"
        pagination={false}
        scroll={{ y: 'calc(100vh - 300px)' }}
        rowClassName={(r) => r.blocked ? 'bg-red-50' : ''}
      />
    </div>
  )
}
