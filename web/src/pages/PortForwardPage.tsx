import { useCallback, useState } from 'react'
import { Table, Button, Popconfirm, message, Space } from 'antd'
import { PlusOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { usePolling } from '../hooks/usePolling'
import { getPortForwards, removePortForward } from '../api/client'
import type { PortForward } from '../api/types'
import AddPortForwardModal from '../components/nat/AddPortForwardModal'

export default function PortForwardPage() {
  const fetcher = useCallback(() => getPortForwards(), [])
  const { data, refresh } = usePolling(fetcher, 30000)
  const [modalOpen, setModalOpen] = useState(false)

  const handleDelete = async (name: string) => {
    try {
      await removePortForward(name)
      message.success('Port forward removed')
      refresh()
    } catch {
      message.error('Failed to remove port forward')
    }
  }

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Protocol', dataIndex: 'protocol', key: 'protocol', render: (v: string) => v.toUpperCase() },
    { title: 'External Port', dataIndex: 'externalPort', key: 'ext' },
    {
      title: 'Internal Target',
      key: 'internal',
      render: (_: unknown, r: PortForward) => `${r.internalIp}:${r.internalPort}`,
    },
    {
      title: 'Action',
      key: 'action',
      render: (_: unknown, r: PortForward) => (
        <Popconfirm title="Remove this rule?" onConfirm={() => handleDelete(r.name)}>
          <Button danger icon={<DeleteOutlined />} size="small">Delete</Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <Space className="mb-4">
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          Add Rule
        </Button>
        <Button icon={<ReloadOutlined />} onClick={refresh}>Refresh</Button>
      </Space>
      <Table dataSource={data || []} columns={columns} rowKey="name" size="small" pagination={false} />
      <AddPortForwardModal open={modalOpen} onClose={() => setModalOpen(false)} onSuccess={refresh} />
    </div>
  )
}
