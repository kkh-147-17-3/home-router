import { useState, useRef, useEffect } from 'react'
import { Select, Button, Space, Switch, Tag } from 'antd'
import { PauseOutlined, CaretRightOutlined, ClearOutlined } from '@ant-design/icons'
import { useSSE } from '../../hooks/useSSE'
import type { LogEntry } from '../../api/types'

const units = [
  { value: 'home-router', label: 'home-router' },
  { value: 'systemd-networkd', label: 'systemd-networkd' },
]

const priorityColor: Record<string, string> = {
  emerg: '#ff4d4f', alert: '#ff4d4f', crit: '#ff4d4f', err: '#ff4d4f',
  warning: '#faad14', notice: '#1677ff', info: '#52c41a', debug: '#8c8c8c',
}

export default function LiveLogStream() {
  const [unit, setUnit] = useState('home-router')
  const [enabled, setEnabled] = useState(true)
  const [autoScroll, setAutoScroll] = useState(true)
  const containerRef = useRef<HTMLDivElement>(null)

  const { data: logs, connected, clear } = useSSE<LogEntry>(
    `/api/sse/system-logs?unit=${unit}`,
    { enabled },
  )

  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  return (
    <div>
      <Space className="mb-4" wrap>
        <Select value={unit} onChange={(v) => { setUnit(v); clear() }} options={units} style={{ width: 180 }} />
        <Button
          icon={enabled ? <PauseOutlined /> : <CaretRightOutlined />}
          onClick={() => setEnabled(!enabled)}
        >
          {enabled ? 'Pause' : 'Resume'}
        </Button>
        <Button icon={<ClearOutlined />} onClick={clear}>Clear</Button>
        <span>Auto-scroll: <Switch checked={autoScroll} onChange={setAutoScroll} size="small" /></span>
        <Tag color={connected ? 'green' : 'red'}>{connected ? 'Connected' : 'Disconnected'}</Tag>
      </Space>
      <div
        ref={containerRef}
        className="bg-gray-900 text-gray-100 p-4 rounded font-mono text-xs overflow-auto"
        style={{ height: 'calc(100vh - 260px)', minHeight: 400 }}
      >
        {logs.map((entry, i) => (
          <div key={i} className="leading-5">
            <span className="text-gray-500">{entry.timestamp ? new Date(entry.timestamp).toLocaleTimeString() : ''}</span>
            {' '}
            <span style={{ color: priorityColor[entry.priority] || '#8c8c8c' }}>[{entry.priority}]</span>
            {' '}
            <span>{entry.message}</span>
          </div>
        ))}
        {logs.length === 0 && <span className="text-gray-500">Waiting for log entries...</span>}
      </div>
    </div>
  )
}
