import { useCallback } from 'react'
import { Row, Col, Spin } from 'antd'
import { usePolling } from '../hooks/usePolling'
import { getDashboard, getDNSStats } from '../api/client'
import StatsCard from '../components/dashboard/StatsCard'
import HourlyQueryChart from '../components/dashboard/HourlyQueryChart'
import BlockedPieChart from '../components/dashboard/BlockedPieChart'
import TopBlockedChart from '../components/dashboard/TopBlockedChart'
import TopClientsChart from '../components/dashboard/TopClientsChart'

export default function DashboardPage() {
  const fetchDashboard = useCallback(() => getDashboard(), [])
  const fetchStats = useCallback(() => getDNSStats(), [])
  const { data: dash, loading } = usePolling(fetchDashboard, 10000)
  const { data: stats } = usePolling(fetchStats, 10000)

  if (loading || !dash) return <Spin className="flex justify-center mt-20" size="large" />

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={12} sm={6}>
          <StatsCard title="WAN IP" value={dash.wanIp || 'N/A'} />
        </Col>
        <Col xs={12} sm={6}>
          <StatsCard title="Active Leases" value={dash.activeLeases} />
        </Col>
        <Col xs={12} sm={6}>
          <StatsCard title="DNS Queries (24h)" value={dash.totalQueries} />
        </Col>
        <Col xs={12} sm={6}>
          <StatsCard title="Block Rate" value={`${dash.blockRate.toFixed(1)}%`} />
        </Col>
      </Row>

      {stats && (
        <>
          <Row gutter={[16, 16]} className="mt-4">
            <Col xs={24} md={16}>
              <HourlyQueryChart hourly={stats.hourly} />
            </Col>
            <Col xs={24} md={8}>
              <BlockedPieChart total={stats.totalQueries} blocked={stats.blockedQueries} />
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
        </>
      )}
    </div>
  )
}
