export interface DashboardData {
  wan_ip: string
  active_leases: number
  total_queries: number
  blocked_queries: number
  block_rate: number
  cache_hit_ratio: number
  dns_enabled: boolean
}

export interface LeaseInfo {
  mac: string
  ip: string
  hostname?: string
  expired_at: string
  static: boolean
}

export interface PoolInfo {
  range_start: string
  range_end: string
  total_leases: number
  static_leases: number
  declined_ips: number
}

export interface QueryEntry {
  timestamp: string
  client_ip: string
  domain: string
  query_type: string
  blocked: boolean
  cached: boolean
  response_time_ms: number
}

export interface QueryLogStats {
  total_queries: number
  blocked_queries: number
  cached_queries: number
  block_percentage: number
  top_blocked: TopEntry[]
  top_clients: TopEntry[]
  hourly: Record<string, number>
}

export interface TopEntry {
  name: string
  count: number
}

export interface CacheStats {
  size: number
  max_size: number
  hits: number
  misses: number
  hit_ratio: number
}

export interface BlockerStats {
  total_domains: number
  sources: number
  whitelist: number
}

export interface WhitelistResponse {
  entries: string[]
  sources: string[]
}

export interface PortForward {
  name: string
  protocol: string
  external_port: number
  internal_ip: string
  internal_port: number
}

export interface WANInfo {
  ip: string
  interface: string
  mac: string
}

export interface LANInfo {
  subnet: string
  interface: string
  mac: string
  gateway: string
}

export interface UptimeInfo {
  uptime_seconds: number
  start_time: string
  uptime: string
}

export interface SystemConfig {
  config: string
}

export interface LogEntry {
  timestamp: string
  priority: string
  unit: string
  message: string
  pid?: number
}
