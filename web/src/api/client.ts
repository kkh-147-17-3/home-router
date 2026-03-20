import axios from 'axios'
import type {
  DashboardData, LeaseInfo, PoolInfo, QueryEntry, QueryLogStats,
  CacheStats, BlockerStats, WhitelistResponse, PortForward,
  WANInfo, LANInfo, UptimeInfo, SystemConfig, LogEntry,
  DDNSStatus, DDNSConfigRequest, AccessEntry, AccessStats,
  TrafficSummary, ConnEntry,
} from './types'

const api = axios.create({ baseURL: '/api' })

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      window.location.href = '/login'
    }
    return Promise.reject(err)
  },
)

// Auth
export const login = (password: string) =>
  api.post('/auth/login', { password })

// Dashboard
export const getDashboard = () =>
  api.get<DashboardData>('/dashboard').then(r => r.data)

// DHCP
export const getDHCPLeases = () =>
  api.get<LeaseInfo[]>('/dhcp/leases').then(r => r.data)

export const getDHCPPool = () =>
  api.get<PoolInfo>('/dhcp/pool').then(r => r.data)

export const getDHCPStaticLeases = () =>
  api.get<LeaseInfo[]>('/dhcp/static-leases').then(r => r.data)

export const addStaticLease = (data: { name: string; mac: string; ip: string }) =>
  api.post('/dhcp/static-leases', data)

export const removeStaticLease = (mac: string) =>
  api.delete(`/dhcp/static-leases/${encodeURIComponent(mac)}`)

// DNS
export const getDNSStats = () =>
  api.get<QueryLogStats>('/dns/stats').then(r => r.data)

export const getDNSQueryLog = (params?: { limit?: number; blocked?: boolean; domain?: string; client?: string }) =>
  api.get<QueryEntry[]>('/dns/querylog', { params }).then(r => r.data)

export const getDNSCacheStats = () =>
  api.get<CacheStats>('/dns/cache/stats').then(r => r.data)

export const getDNSBlockerStats = () =>
  api.get<BlockerStats>('/dns/blocker/stats').then(r => r.data)

export const reloadBlocker = () =>
  api.post('/dns/blocker/reload')

export const getWhitelist = () =>
  api.get<WhitelistResponse>('/dns/blocker/whitelist').then(r => r.data)

export const addWhitelist = (domain: string) =>
  api.post('/dns/blocker/whitelist', { domain })

export const removeWhitelist = (domain: string) =>
  api.delete(`/dns/blocker/whitelist/${encodeURIComponent(domain)}`)

// NAT
export const getPortForwards = () =>
  api.get<PortForward[]>('/nat/port-forwards').then(r => r.data)

export const addPortForward = (data: PortForward) =>
  api.post('/nat/port-forwards', data)

export const removePortForward = (name: string) =>
  api.delete(`/nat/port-forwards/${encodeURIComponent(name)}`)

// Network
export const getWANInfo = () =>
  api.get<WANInfo>('/network/wan').then(r => r.data)

export const getLANInfo = () =>
  api.get<LANInfo>('/network/lan').then(r => r.data)

// System
export const getUptime = () =>
  api.get<UptimeInfo>('/system/uptime').then(r => r.data)

export const getSystemConfig = () =>
  api.get<SystemConfig>('/system/config').then(r => r.data)

export const getSystemLogs = (params?: { unit?: string; lines?: string; priority?: string; since?: string; grep?: string }) =>
  api.get<LogEntry[]>('/system/logs', { params }).then(r => r.data)

// DDNS
export const getDDNSStatus = () =>
  api.get<DDNSStatus>('/ddns/status').then(r => r.data)

export const updateDDNS = () =>
  api.post('/ddns/update')

export const updateDDNSConfig = (data: DDNSConfigRequest) =>
  api.post('/ddns/config', data)

// Monitor
export const getAccessLog = (params?: Record<string, string>) =>
  api.get<AccessEntry[]>('/monitor/access-log', { params }).then(r => r.data)

export const getMonitorStats = () =>
  api.get<AccessStats>('/monitor/stats').then(r => r.data)

export const getTraffic = () =>
  api.get<TrafficSummary>('/monitor/traffic').then(r => r.data)

export const getConnections = (host?: string) =>
  api.get<ConnEntry[]>('/monitor/connections', { params: host ? { host } : {} }).then(r => r.data)
