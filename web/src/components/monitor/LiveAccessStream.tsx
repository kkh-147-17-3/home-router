import { Tag, Badge, Button, Table } from 'antd'
import { ClearOutlined } from '@ant-design/icons'
import { useSSE } from '../../hooks/useSSE'
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

export default function LiveAccessStream() {
  const { data, connected, clear } = useSSE<AccessEntry>('/api/sse/access-log', { maxItems: 200 })

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
        <span>
          {r.sourceIp}
          <CountryFlag code={r.countryCode} country={r.country} />
        </span>
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

  const reversed = [...data].reverse()

  return (
    <div>
      <div className="mb-4 flex items-center gap-4">
        <Badge status={connected ? 'success' : 'error'}
          text={connected ? 'Connected' : 'Disconnected'} />
        <span className="text-gray-500">{data.length} events</span>
        <Button icon={<ClearOutlined />} size="small" onClick={clear}>Clear</Button>
      </div>
      <Table
        dataSource={reversed}
        columns={columns}
        rowKey={(_, i) => String(i)}
        size="small"
        pagination={false}
        scroll={{ y: 500 }}
      />
    </div>
  )
}
