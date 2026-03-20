import { useCallback, useState } from 'react'
import { Card, Statistic, Row, Col, Tabs, Table, Button, Tag } from 'antd'
import { AlertOutlined, UserOutlined, StopOutlined, ReloadOutlined, CloudDownloadOutlined, CloudUploadOutlined } from '@ant-design/icons'
import { usePolling } from '../hooks/usePolling'
import { getMonitorStats, getTraffic, getConnections } from '../api/client'
import type { HostTraffic, EndpointStat, ConnEntry } from '../api/types'
import AccessLogTable from '../components/monitor/AccessLogTable'
import LiveAccessStream from '../components/monitor/LiveAccessStream'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function TrafficTab() {
  const fetcher = useCallback(() => getTraffic(), [])
  const { data: traffic, refresh } = usePolling(fetcher, 5000)

  const hostColumns = [
    {
      title: 'Host',
      key: 'host',
      render: (_: unknown, r: HostTraffic) => r.hostname ? `${r.hostname} (${r.ip})` : r.ip,
    },
    {
      title: 'Sent',
      dataIndex: 'bytesSent',
      key: 'sent',
      render: (v: number) => formatBytes(v),
      sorter: (a: HostTraffic, b: HostTraffic) => a.bytesSent - b.bytesSent,
    },
    {
      title: 'Received',
      dataIndex: 'bytesRecv',
      key: 'recv',
      render: (v: number) => formatBytes(v),
      sorter: (a: HostTraffic, b: HostTraffic) => a.bytesRecv - b.bytesRecv,
    },
    {
      title: 'Total',
      key: 'total',
      render: (_: unknown, r: HostTraffic) => formatBytes(r.bytesSent + r.bytesRecv),
      sorter: (a: HostTraffic, b: HostTraffic) =>
        (a.bytesSent + a.bytesRecv) - (b.bytesSent + b.bytesRecv),
      defaultSortOrder: 'descend' as const,
    },
    { title: 'Connections', dataIndex: 'connections', key: 'conns' },
  ]

  const destColumns = [
    {
      title: 'Destination',
      key: 'ip',
      render: (_: unknown, r: EndpointStat) => {
        const label = r.domain || r.org
        return label
          ? <span title={r.ip}>{label}</span>
          : r.ip
      },
    },
    { title: 'Port', dataIndex: 'port', key: 'port' },
    {
      title: 'Protocol',
      dataIndex: 'protocol',
      key: 'proto',
      render: (v: string) => v.toUpperCase(),
    },
    {
      title: 'Sent',
      dataIndex: 'bytesSent',
      key: 'sent',
      render: (v: number) => formatBytes(v),
    },
    {
      title: 'Received',
      dataIndex: 'bytesRecv',
      key: 'recv',
      render: (v: number) => formatBytes(v),
    },
    {
      title: 'Total',
      key: 'total',
      render: (_: unknown, r: EndpointStat) => formatBytes(r.bytesSent + r.bytesRecv),
    },
    { title: 'Connections', dataIndex: 'connections', key: 'conns' },
  ]

  const expandedRowRender = (record: HostTraffic) => (
    <Table
      dataSource={record.topDestinations || []}
      columns={destColumns}
      rowKey={(_, i) => String(i)}
      size="small"
      pagination={false}
    />
  )

  return (
    <div>
      <Row gutter={16} className="mb-4">
        <Col span={8}>
          <Card size="small">
            <Statistic title="Total Sent" value={formatBytes(traffic?.totalSent ?? 0)}
              prefix={<CloudUploadOutlined />} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title="Total Received" value={formatBytes(traffic?.totalRecv ?? 0)}
              prefix={<CloudDownloadOutlined />} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title="Active Connections" value={traffic?.totalConnections ?? 0} />
          </Card>
        </Col>
      </Row>

      <div className="mb-2 flex justify-between items-center">
        <h4 className="m-0 font-semibold">Per-Host Traffic</h4>
        <Button icon={<ReloadOutlined />} size="small" onClick={refresh}>Refresh</Button>
      </div>
      <Table
        dataSource={traffic?.hosts || []}
        columns={hostColumns}
        rowKey="ip"
        size="small"
        expandable={{ expandedRowRender }}
        pagination={false}
      />

      <h4 className="mt-6 mb-2 font-semibold">Top Destinations (All Hosts)</h4>
      <Table
        dataSource={traffic?.topDestinations || []}
        columns={destColumns}
        rowKey={(_, i) => String(i)}
        size="small"
        pagination={false}
      />
    </div>
  )
}

function ConnectionsTab() {
  const fetcher = useCallback(() => getConnections(), [])
  const { data, refresh } = usePolling(fetcher, 5000)

  const columns = [
    { title: 'Source IP', dataIndex: 'srcIp', key: 'src' },
    {
      title: 'Destination',
      key: 'dst',
      render: (_: unknown, r: ConnEntry) => {
        const label = r.dstDomain || r.dstOrg
        return label
          ? <span title={r.dstIp}>{label}</span>
          : r.dstIp
      },
    },
    { title: 'Dest Port', dataIndex: 'dstPort', key: 'dport' },
    {
      title: 'Protocol',
      dataIndex: 'protocol',
      key: 'proto',
      render: (v: string) => v.toUpperCase(),
    },
    {
      title: 'State',
      dataIndex: 'state',
      key: 'state',
      render: (v: string) => (
        <Tag color={v === 'ESTABLISHED' ? 'green' : v === 'TIME_WAIT' ? 'orange' : 'default'}>
          {v}
        </Tag>
      ),
    },
    {
      title: 'Sent',
      dataIndex: 'bytesSent',
      key: 'sent',
      render: (v: number) => formatBytes(v),
      sorter: (a: ConnEntry, b: ConnEntry) => a.bytesSent - b.bytesSent,
    },
    {
      title: 'Received',
      dataIndex: 'bytesRecv',
      key: 'recv',
      render: (v: number) => formatBytes(v),
      sorter: (a: ConnEntry, b: ConnEntry) => a.bytesRecv - b.bytesRecv,
    },
  ]

  return (
    <div>
      <div className="mb-4">
        <Button icon={<ReloadOutlined />} size="small" onClick={refresh}>Refresh</Button>
      </div>
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

export default function SecurityPage() {
  const statsFetcher = useCallback(() => getMonitorStats(), [])
  const { data: stats } = usePolling(statsFetcher, 5000)
  const [activeTab, setActiveTab] = useState('traffic')

  return (
    <div className="space-y-4">
      <Row gutter={16}>
        <Col span={8}>
          <Card>
            <Statistic
              title="WAN Events (24h)"
              value={stats?.totalEvents ?? 0}
              prefix={<AlertOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Unique Source IPs"
              value={stats?.uniqueSourceIps ?? 0}
              prefix={<UserOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Top Attacker"
              value={stats?.topSourceIps?.[0]?.name ?? '-'}
              suffix={stats?.topSourceIps?.[0] ? `(${stats.topSourceIps[0].count})` : ''}
              prefix={<StopOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Tabs activeKey={activeTab} onChange={setActiveTab} items={[
          {
            key: 'traffic',
            label: 'Traffic',
            children: <TrafficTab />,
          },
          {
            key: 'connections',
            label: 'Connections',
            children: <ConnectionsTab />,
          },
          {
            key: 'live',
            label: 'Live Monitor',
            children: <LiveAccessStream />,
          },
          {
            key: 'log',
            label: 'Access Log',
            children: <AccessLogTable />,
          },
        ]} />
      </Card>
    </div>
  )
}
