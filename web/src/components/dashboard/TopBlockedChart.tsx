import { Card } from 'antd'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import type { TopEntry } from '../../api/types'

export default function TopBlockedChart({ data }: { data: TopEntry[] }) {
  return (
    <Card title="Top Blocked Domains">
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={data} layout="vertical" margin={{ left: 100 }}>
          <XAxis type="number" fontSize={12} />
          <YAxis type="category" dataKey="name" fontSize={11} width={100} />
          <Tooltip />
          <Bar dataKey="count" fill="#ff4d4f" />
        </BarChart>
      </ResponsiveContainer>
    </Card>
  )
}
