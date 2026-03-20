import { Card, List, Button, message } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { reloadBlocker } from '../../api/client'

interface Props {
  sources: string[]
}

export default function BlocklistView({ sources }: Props) {
  const handleReload = async () => {
    try {
      await reloadBlocker()
      message.success('Blocklist reload started')
    } catch {
      message.error('Failed to reload blocklist')
    }
  }

  return (
    <Card
      title="Blocklist Sources"
      extra={<Button icon={<ReloadOutlined />} onClick={handleReload}>Reload</Button>}
    >
      <List
        dataSource={sources}
        renderItem={(item) => (
          <List.Item>
            <code className="text-sm break-all">{item}</code>
          </List.Item>
        )}
        locale={{ emptyText: 'No blocklist sources configured' }}
      />
    </Card>
  )
}
