import { useCallback, useState } from 'react'
import { Tabs, Spin } from 'antd'
import { usePolling } from '../hooks/usePolling'
import { getDHCPLeases, getDHCPPool, getDHCPStaticLeases } from '../api/client'
import ActiveLeasesTable from '../components/dhcp/ActiveLeasesTable'
import StaticLeasesTable from '../components/dhcp/StaticLeasesTable'
import PoolInfoCard from '../components/dhcp/PoolInfoCard'

export default function DHCPPage() {
  const fetchLeases = useCallback(() => getDHCPLeases(), [])
  const fetchPool = useCallback(() => getDHCPPool(), [])
  const fetchStatic = useCallback(() => getDHCPStaticLeases(), [])
  const { data: leases, loading, refresh: refreshLeases } = usePolling(fetchLeases, 10000)
  const { data: pool } = usePolling(fetchPool, 10000)
  const { data: staticLeases, refresh: refreshStatic } = usePolling(fetchStatic, 10000)
  const [tab, setTab] = useState('leases')

  if (loading) return <Spin className="flex justify-center mt-20" size="large" />

  return (
    <Tabs activeKey={tab} onChange={setTab} items={[
      {
        key: 'leases',
        label: 'Active Leases',
        children: <ActiveLeasesTable leases={leases || []} onRefresh={refreshLeases} />,
      },
      {
        key: 'static',
        label: 'Static Leases',
        children: <StaticLeasesTable leases={staticLeases || []} onRefresh={refreshStatic} />,
      },
      {
        key: 'pool',
        label: 'Pool Settings',
        children: pool ? <PoolInfoCard pool={pool} /> : null,
      },
    ]} />
  )
}
