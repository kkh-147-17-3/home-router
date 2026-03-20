import { Card } from 'antd'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import type { TopEntry } from '../../api/types'

export default function TopClientsChart({ data }: { data: TopEntry[] }) {
  return (
    <Card title="Top Clients">
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={data} layout="vertical" margin={{ left: 100 }}>
          <XAxis type="number" fontSize={12} />
          <YAxis type="category" dataKey="name" fontSize={11} width={100} />
          <Tooltip />
          <Bar dataKey="count" fill="#1677ff" />
        </BarChart>
      </ResponsiveContainer>
    </Card>
  )
}
