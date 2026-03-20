import { Layout, Menu } from 'antd'
import {
  DashboardOutlined, WifiOutlined, SafetyOutlined,
  SwapOutlined, GlobalOutlined, SettingOutlined, FileTextOutlined,
} from '@ant-design/icons'
import { useNavigate, useLocation } from 'react-router-dom'

const { Sider } = Layout

const menuItems = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: 'Dashboard' },
  { key: '/dhcp', icon: <WifiOutlined />, label: 'DHCP' },
  { key: '/dns', icon: <SafetyOutlined />, label: 'DNS' },
  { key: '/port-forward', icon: <SwapOutlined />, label: 'Port Forward' },
  { key: '/network', icon: <GlobalOutlined />, label: 'Network' },
  { key: '/system', icon: <SettingOutlined />, label: 'System' },
  { key: '/logs', icon: <FileTextOutlined />, label: 'Logs' },
]

export default function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()

  return (
    <Sider breakpoint="lg" collapsedWidth="60" className="min-h-screen">
      <div className="text-white text-center py-4 text-lg font-bold">
        Home Router
      </div>
      <Menu
        theme="dark"
        mode="inline"
        selectedKeys={[location.pathname]}
        items={menuItems}
        onClick={({ key }) => navigate(key)}
      />
    </Sider>
  )
}
