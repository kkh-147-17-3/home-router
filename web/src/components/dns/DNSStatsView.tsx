import { Row, Col, Card, Statistic } from 'antd'
import type { QueryLogStats, CacheStats, BlockerStats } from '../../api/types'
import HourlyQueryChart from '../dashboard/HourlyQueryChart'
import TopBlockedChart from '../dashboard/TopBlockedChart'
import TopClientsChart from '../dashboard/TopClientsChart'

interface Props {
  stats: QueryLogStats
  cacheStats: CacheStats
  blockerStats: BlockerStats
}

export default function DNSStatsView({ stats, cacheStats, blockerStats }: Props) {
  if (!stats || !cacheStats || !blockerStats) return null

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={12} sm={6}>
          <Card><Statistic title="Total Queries (24h)" value={stats.totalQueries} /></Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card><Statistic title="Blocked" value={stats.blockedQueries} valueStyle={{ color: '#ff4d4f' }} /></Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card><Statistic title="Cache Hit Ratio" value={`${cacheStats.hitRatio.toFixed(1)}%`} /></Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card><Statistic title="Blocked Domains" value={blockerStats.totalDomains.toLocaleString()} /></Card>
        </Col>
      </Row>
      <Row gutter={[16, 16]} className="mt-4">
        <Col span={24}>
          <HourlyQueryChart hourly={stats.hourly} />
        </Col>
      </Row>
      <Row gutter={[16, 16]} className="mt-4">
        <Col xs={24} md={12}>
          <TopBlockedChart data={stats.topBlocked} />
        </Col>
        <Col xs={24} md={12}>
          <TopClientsChart data={stats.topClients} />
        </Col>
      </Row>
    </div>
  )
}
