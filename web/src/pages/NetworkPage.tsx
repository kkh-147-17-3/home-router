import { useCallback } from 'react'
import { Row, Col, Card, Descriptions, Spin } from 'antd'
import { usePolling } from '../hooks/usePolling'
import { getWANInfo, getLANInfo } from '../api/client'

export default function NetworkPage() {
  const fetchWAN = useCallback(() => getWANInfo(), [])
  const fetchLAN = useCallback(() => getLANInfo(), [])
  const { data: wan, loading } = usePolling(fetchWAN, 10000)
  const { data: lan } = usePolling(fetchLAN, 10000)

  if (loading) return <Spin className="flex justify-center mt-20" size="large" />

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} md={12}>
        <Card title="WAN">
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="IP">{wan?.ip || 'N/A'}</Descriptions.Item>
            <Descriptions.Item label="Interface">{wan?.interface}</Descriptions.Item>
            <Descriptions.Item label="MAC">{wan?.mac}</Descriptions.Item>
          </Descriptions>
        </Card>
      </Col>
      <Col xs={24} md={12}>
        <Card title="LAN">
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="Subnet">{lan?.subnet}</Descriptions.Item>
            <Descriptions.Item label="Gateway">{lan?.gateway}</Descriptions.Item>
            <Descriptions.Item label="Interface">{lan?.interface}</Descriptions.Item>
            <Descriptions.Item label="MAC">{lan?.mac}</Descriptions.Item>
          </Descriptions>
        </Card>
      </Col>
    </Row>
  )
}
