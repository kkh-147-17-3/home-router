import { Table, Tag, Button } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import type { LeaseInfo } from '../../api/types'

interface Props {
  leases: LeaseInfo[]
  onRefresh: () => void
}

export default function ActiveLeasesTable({ leases, onRefresh }: Props) {
  const columns = [
    { title: 'MAC', dataIndex: 'mac', key: 'mac', sorter: (a: LeaseInfo, b: LeaseInfo) => a.mac.localeCompare(b.mac) },
    { title: 'IP', dataIndex: 'ip', key: 'ip', sorter: (a: LeaseInfo, b: LeaseInfo) => a.ip.localeCompare(b.ip) },
    { title: 'Hostname', dataIndex: 'hostname', key: 'hostname', render: (v: string) => v || '-' },
    {
      title: 'Expires',
      dataIndex: 'expiredAt',
      key: 'expiredAt',
      render: (v: string) => new Date(v).toLocaleString(),
      sorter: (a: LeaseInfo, b: LeaseInfo) => new Date(a.expiredAt).getTime() - new Date(b.expiredAt).getTime(),
    },
    {
      title: 'Type',
      dataIndex: 'static',
      key: 'static',
      render: (v: boolean) => v ? <Tag color="blue">Static</Tag> : <Tag color="green">Dynamic</Tag>,
    },
  ]

  return (
    <div>
      <div className="mb-4">
        <Button icon={<ReloadOutlined />} onClick={onRefresh}>Refresh</Button>
      </div>
      <Table dataSource={leases} columns={columns} rowKey="mac" size="small" pagination={false} />
    </div>
  )
}
