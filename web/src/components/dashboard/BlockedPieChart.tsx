import { Card } from 'antd'
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts'

interface Props {
  total: number
  blocked: number
}

const COLORS = ['#52c41a', '#ff4d4f']

export default function BlockedPieChart({ total, blocked }: Props) {
  const allowed = total - blocked
  const data = [
    { name: 'Allowed', value: allowed },
    { name: 'Blocked', value: blocked },
  ]

  return (
    <Card title="Blocked vs Allowed">
      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie data={data} cx="50%" cy="50%" innerRadius={50} outerRadius={80} dataKey="value" label>
            {data.map((_, i) => <Cell key={i} fill={COLORS[i]} />)}
          </Pie>
          <Tooltip />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </Card>
  )
}
