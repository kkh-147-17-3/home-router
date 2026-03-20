import { useState } from 'react'
import { Table, Button, Input, Space, Popconfirm, message } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { addWhitelist, removeWhitelist } from '../../api/client'

interface Props {
  entries: string[]
  onRefresh: () => void
}

export default function WhitelistTable({ entries, onRefresh }: Props) {
  const [domain, setDomain] = useState('')

  const handleAdd = async () => {
    if (!domain.trim()) return
    try {
      await addWhitelist(domain.trim())
      message.success('Domain added to whitelist')
      setDomain('')
      onRefresh()
    } catch {
      message.error('Failed to add domain')
    }
  }

  const handleRemove = async (d: string) => {
    try {
      await removeWhitelist(d)
      message.success('Domain removed from whitelist')
      onRefresh()
    } catch {
      message.error('Failed to remove domain')
    }
  }

  const data = entries.map(e => ({ domain: e }))

  return (
    <div>
      <Space className="mb-4">
        <Input
          placeholder="example.com"
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
          onPressEnter={handleAdd}
          style={{ width: 300 }}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>Add</Button>
      </Space>
      <Table
        dataSource={data}
        rowKey="domain"
        size="small"
        pagination={false}
        columns={[
          { title: 'Domain', dataIndex: 'domain', key: 'domain' },
          {
            title: 'Action',
            key: 'action',
            width: 100,
            render: (_, record: { domain: string }) => (
              <Popconfirm title="Remove from whitelist?" onConfirm={() => handleRemove(record.domain)}>
                <Button danger icon={<DeleteOutlined />} size="small">Delete</Button>
              </Popconfirm>
            ),
          },
        ]}
      />
    </div>
  )
}
