import { Card } from 'antd'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

interface Props {
  hourly: Record<string, number>
}

export default function HourlyQueryChart({ hourly }: Props) {
  const data = Array.from({ length: 24 }, (_, i) => ({
    hour: `${i}:00`,
    queries: hourly[String(i)] || 0,
  }))

  return (
    <Card title="Hourly Query Distribution">
      <ResponsiveContainer width="100%" height={250}>
        <AreaChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="hour" fontSize={12} />
          <YAxis fontSize={12} />
          <Tooltip />
          <Area type="monotone" dataKey="queries" stroke="#1677ff" fill="#1677ff" fillOpacity={0.3} />
        </AreaChart>
      </ResponsiveContainer>
    </Card>
  )
}
