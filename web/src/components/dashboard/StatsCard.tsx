import { Card, Statistic } from 'antd'

interface Props {
  title: string
  value: string | number
}

export default function StatsCard({ title, value }: Props) {
  return (
    <Card>
      <Statistic title={title} value={value} />
    </Card>
  )
}
