import { useState } from 'react'
import { Table, Button, Space, Popconfirm, message } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import type { LeaseInfo } from '../../api/types'
import { removeStaticLease } from '../../api/client'
import AddStaticLeaseModal from './AddStaticLeaseModal'

interface Props {
  leases: LeaseInfo[]
  onRefresh: () => void
}

export default function StaticLeasesTable({ leases, onRefresh }: Props) {
  const [modalOpen, setModalOpen] = useState(false)

  const handleDelete = async (mac: string) => {
    try {
      await removeStaticLease(mac)
      message.success('Static lease removed')
      onRefresh()
    } catch {
      message.error('Failed to remove static lease')
    }
  }

  const columns = [
    { title: 'MAC', dataIndex: 'mac', key: 'mac' },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    {
      title: 'Action',
      key: 'action',
      render: (_: unknown, record: LeaseInfo) => (
        <Popconfirm title="Remove this static lease?" onConfirm={() => handleDelete(record.mac)}>
          <Button danger icon={<DeleteOutlined />} size="small">Delete</Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <div className="mb-4">
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          Add Static Lease
        </Button>
      </div>
      <Table dataSource={leases} columns={columns} rowKey="mac" size="small" pagination={false} />
      <AddStaticLeaseModal open={modalOpen} onClose={() => setModalOpen(false)} onSuccess={onRefresh} />
    </div>
  )
}
