import { Card } from 'antd'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import type { TopEntry } from '../../api/types'

export default function TopClientsChart({ data }: { data: TopEntry[] }) {
  const chartData = data.map(d => ({
    ...d,
    displayName: d.label ? `${d.label} (${d.name})` : d.name,
  }))

  return (
    <Card title="Top Clients">
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData} layout="vertical" margin={{ left: 120 }}>
          <XAxis type="number" fontSize={12} />
          <YAxis type="category" dataKey="displayName" fontSize={11} width={120} />
          <Tooltip />
          <Bar dataKey="count" fill="#1677ff" />
        </BarChart>
      </ResponsiveContainer>
    </Card>
  )
}
