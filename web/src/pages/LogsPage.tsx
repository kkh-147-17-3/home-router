import { useState } from 'react'
import { Tabs } from 'antd'
import LogViewer from '../components/logs/LogViewer'
import LiveLogStream from '../components/logs/LiveLogStream'
import LiveDNSStream from '../components/logs/LiveDNSStream'

export default function LogsPage() {
  const [tab, setTab] = useState('logs')

  return (
    <Tabs activeKey={tab} onChange={setTab} items={[
      { key: 'logs', label: 'Service Logs', children: <LogViewer /> },
      { key: 'live', label: 'Live Stream', children: <LiveLogStream /> },
      { key: 'dns-live', label: 'DNS Live Query', children: <LiveDNSStream /> },
    ]} />
  )
}
