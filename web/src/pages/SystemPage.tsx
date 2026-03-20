import { useCallback } from 'react'
import { Row, Col, Card, Statistic, Spin } from 'antd'
import { usePolling } from '../hooks/usePolling'
import { getUptime, getSystemConfig } from '../api/client'

export default function SystemPage() {
  const fetchUptime = useCallback(() => getUptime(), [])
  const fetchConfig = useCallback(() => getSystemConfig(), [])
  const { data: uptime, loading } = usePolling(fetchUptime, 5000)
  const { data: config } = usePolling(fetchConfig, 60000)

  if (loading) return <Spin className="flex justify-center mt-20" size="large" />

  return (
    <div>
      <Row gutter={[16, 16]} className="mb-4">
        <Col xs={24} sm={12}>
          <Card>
            <Statistic title="Uptime" value={uptime?.uptime || 'N/A'} />
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card>
            <Statistic title="Start Time" value={uptime?.start_time ? new Date(uptime.start_time).toLocaleString() : 'N/A'} />
          </Card>
        </Col>
      </Row>
      <Card title="Configuration (read-only)">
        <pre className="bg-gray-900 text-green-400 p-4 rounded overflow-auto max-h-[600px] text-sm">
          {config?.config || 'Loading...'}
        </pre>
      </Card>
    </div>
  )
}
