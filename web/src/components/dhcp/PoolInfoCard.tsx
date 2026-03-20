import { Card, Descriptions } from 'antd'
import type { PoolInfo } from '../../api/types'

export default function PoolInfoCard({ pool }: { pool: PoolInfo }) {
  return (
    <Card>
      <Descriptions bordered column={1}>
        <Descriptions.Item label="Range Start">{pool.rangeStart}</Descriptions.Item>
        <Descriptions.Item label="Range End">{pool.rangeEnd}</Descriptions.Item>
        <Descriptions.Item label="Total Leases">{pool.totalLeases}</Descriptions.Item>
        <Descriptions.Item label="Static Leases">{pool.staticLeases}</Descriptions.Item>
        <Descriptions.Item label="Declined IPs">{pool.declinedIps}</Descriptions.Item>
      </Descriptions>
    </Card>
  )
}
