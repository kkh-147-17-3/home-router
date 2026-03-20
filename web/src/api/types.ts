export interface DashboardData {
  wanIp: string
  activeLeases: number
  totalQueries: number
  blockedQueries: number
  blockRate: number
  cacheHitRatio: number
  dnsEnabled: boolean
}

export interface LeaseInfo {
  mac: string
  ip: string
  hostname?: string
  expiredAt: string
  static: boolean
}

export interface PoolInfo {
  rangeStart: string
  rangeEnd: string
  totalLeases: number
  staticLeases: number
  declinedIps: number
}

export interface QueryEntry {
  timestamp: string
  clientIp: string
  domain: string
  queryType: string
  blocked: boolean
  cached: boolean
  responseTimeMs: number
}

export interface QueryLogStats {
  totalQueries: number
  blockedQueries: number
  cachedQueries: number
  blockPercentage: number
  topBlocked: TopEntry[]
  topClients: TopEntry[]
  hourly: Record<string, number>
}

export interface TopEntry {
  name: string
  label?: string
  count: number
}

export interface CacheStats {
  size: number
  maxSize: number
  hits: number
  misses: number
  hitRatio: number
}

export interface BlockerStats {
  totalDomains: number
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
  externalPort: number
  internalIp: string
  internalPort: number
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
  uptimeSeconds: number
  startTime: string
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

// DDNS
export interface DDNSStatus {
  enabled: boolean
  provider: string
  domain: string
  lastIp: string
  lastUpdate: string
  lastError: string
}

export interface DDNSConfigRequest {
  enabled: boolean
  provider: string
  domain: string
  token: string
  zoneId?: string
  recordId?: string
  proxied?: boolean
  updateUrl?: string
}

// Monitor
export interface AccessEntry {
  timestamp: string
  sourceIp: string
  country?: string
  countryCode?: string
  destIp?: string
  destPort: number
  portName?: string
  protocol: string
  action: string
  reason: string
}

export interface AccessStats {
  totalEvents: number
  uniqueSourceIps: number
  topSourceIps: TopEntry[]
  topPorts: TopEntry[]
  hourly: Record<string, number>
}

// Traffic
export interface ConnEntry {
  protocol: string
  state: string
  srcIp: string
  dstIp: string
  dstDomain?: string
  srcPort: number
  dstPort: number
  bytesSent: number
  bytesRecv: number
}

export interface EndpointStat {
  ip: string
  domain?: string
  port: number
  protocol: string
  bytesSent: number
  bytesRecv: number
  connections: number
}

export interface HostTraffic {
  ip: string
  hostname?: string
  bytesSent: number
  bytesRecv: number
  connections: number
  topDestinations: EndpointStat[]
}

export interface TrafficSummary {
  totalSent: number
  totalRecv: number
  totalConnections: number
  hosts: HostTraffic[]
  topDestinations: EndpointStat[]
}
