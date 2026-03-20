import { useCallback, useState } from 'react'
import { Tabs, Spin } from 'antd'
import { usePolling } from '../hooks/usePolling'
import { getDNSStats, getDNSCacheStats, getDNSBlockerStats, getWhitelist } from '../api/client'
import QueryLogTable from '../components/dns/QueryLogTable'
import DNSStatsView from '../components/dns/DNSStatsView'
import BlocklistView from '../components/dns/BlocklistView'
import WhitelistTable from '../components/dns/WhitelistTable'

export default function DNSPage() {
  const fetchStats = useCallback(() => getDNSStats(), [])
  const fetchCache = useCallback(() => getDNSCacheStats(), [])
  const fetchBlocker = useCallback(() => getDNSBlockerStats(), [])
  const fetchWhitelist = useCallback(() => getWhitelist(), [])
  const { data: stats, loading } = usePolling(fetchStats, 10000)
  const { data: cacheStats } = usePolling(fetchCache, 10000)
  const { data: blockerStats } = usePolling(fetchBlocker, 10000)
  const { data: whitelist, refresh: refreshWhitelist } = usePolling(fetchWhitelist, 30000)
  const [tab, setTab] = useState('querylog')

  if (loading) return <Spin className="flex justify-center mt-20" size="large" />

  return (
    <Tabs activeKey={tab} onChange={setTab} items={[
      { key: 'querylog', label: 'Query Log', children: <QueryLogTable /> },
      {
        key: 'stats',
        label: 'Statistics',
        children: <DNSStatsView stats={stats!} cacheStats={cacheStats!} blockerStats={blockerStats!} />,
      },
      {
        key: 'blocklist',
        label: 'Blocklists',
        children: <BlocklistView sources={whitelist?.sources || []} />,
      },
      {
        key: 'whitelist',
        label: 'Whitelist',
        children: <WhitelistTable entries={whitelist?.entries || []} onRefresh={refreshWhitelist} />,
      },
    ]} />
  )
}
