import { useState, useEffect } from 'react'
import { Card, Input, Button, Alert, Typography, Spin } from 'antd'
import { LockOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import type { AuthState } from '../hooks/useAuth'

export default function LoginPage({ auth }: { auth: AuthState }) {
  const [password, setPassword] = useState('')
  const navigate = useNavigate()

  useEffect(() => {
    if (auth.authenticated) navigate('/dashboard', { replace: true })
  }, [auth.authenticated, navigate])

  if (auth.loading) {
    return <div className="flex items-center justify-center min-h-screen"><Spin size="large" /></div>
  }

  const handleSubmit = async () => {
    try {
      await auth.login(password)
      navigate('/dashboard', { replace: true })
    } catch { /* error handled in auth */ }
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <Card className="w-96">
        <div className="text-center mb-6">
          <LockOutlined className="text-4xl text-blue-500" />
          <Typography.Title level={3} className="mt-2">Home Router</Typography.Title>
        </div>
        {auth.error && <Alert message={auth.error} type="error" className="mb-4" />}
        <Input.Password
          size="large"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          onPressEnter={handleSubmit}
          className="mb-4"
        />
        <Button type="primary" block size="large" loading={auth.loading} onClick={handleSubmit}>
          Login
        </Button>
      </Card>
    </div>
  )
}
