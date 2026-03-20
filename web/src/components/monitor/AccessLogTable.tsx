import { useCallback, useState } from 'react'
import { Table, Input, Space, Button, Tag } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { usePolling } from '../../hooks/usePolling'
import { getAccessLog } from '../../api/client'
import type { AccessEntry } from '../../api/types'

function CountryFlag({ code, country }: { code?: string; country?: string }) {
  if (!code || code.length !== 2 || code === '??') return null
  return (
    <img
      src={`https://flagcdn.com/16x12/${code.toLowerCase()}.png`}
      alt={country || code}
      title={country || code}
      className="inline-block ml-1 align-middle"
      width={16}
      height={12}
    />
  )
}

export default function AccessLogTable() {
  const [sourceIP, setSourceIP] = useState('')
  const [destPort, setDestPort] = useState('')

  const fetcher = useCallback(() => {
    const params: Record<string, string> = { limit: '200' }
    if (sourceIP) params.source_ip = sourceIP
    if (destPort) params.dest_port = destPort
    return getAccessLog(params)
  }, [sourceIP, destPort])

  const { data, refresh } = usePolling(fetcher, 10000)

  const columns = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'time',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: 'Source',
      key: 'src',
      render: (_: unknown, r: AccessEntry) => (
        <div>
          <span>{r.sourceIp}</span>
          <CountryFlag code={r.countryCode} country={r.country} />
          {r.org && <div className="text-xs text-gray-400">{r.org}</div>}
        </div>
      ),
    },
    {
      title: 'Destination',
      key: 'dest',
      render: (_: unknown, r: AccessEntry) => (
        <span>
          {r.destIp && <span className="mr-1 text-gray-500">{r.destIp}</span>}
          <span>{r.destPort}</span>
          {r.portName && <span className="ml-1 text-gray-400">({r.portName})</span>}
        </span>
      ),
    },
    {
      title: 'Protocol',
      dataIndex: 'protocol',
      key: 'proto',
      width: 80,
      render: (v: string) => v.toUpperCase(),
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
      width: 80,
      render: (v: string) => (
        <Tag color={v === 'DROP' ? 'red' : v === 'ACCEPT' ? 'green' : 'default'}>
          {v}
        </Tag>
      ),
    },
  ]

  return (
    <div>
      <Space className="mb-4">
        <Input placeholder="Filter by Source IP" value={sourceIP}
          onChange={(e) => setSourceIP(e.target.value)} allowClear style={{ width: 200 }} />
        <Input placeholder="Filter by Dest Port" value={destPort}
          onChange={(e) => setDestPort(e.target.value)} allowClear style={{ width: 160 }} />
        <Button icon={<ReloadOutlined />} onClick={refresh}>Refresh</Button>
      </Space>
      <Table
        dataSource={data || []}
        columns={columns}
        rowKey={(_, i) => String(i)}
        size="small"
        pagination={{ pageSize: 50 }}
      />
    </div>
  )
}
