import { Card, Descriptions } from 'antd'
import type { PoolInfo } from '../../api/types'

export default function PoolInfoCard({ pool }: { pool: PoolInfo }) {
  return (
    <Card>
      <Descriptions bordered column={1}>
        <Descriptions.Item label="Range Start">{pool.range_start}</Descriptions.Item>
        <Descriptions.Item label="Range End">{pool.range_end}</Descriptions.Item>
        <Descriptions.Item label="Total Leases">{pool.total_leases}</Descriptions.Item>
        <Descriptions.Item label="Static Leases">{pool.static_leases}</Descriptions.Item>
        <Descriptions.Item label="Declined IPs">{pool.declined_ips}</Descriptions.Item>
      </Descriptions>
    </Card>
  )
}
