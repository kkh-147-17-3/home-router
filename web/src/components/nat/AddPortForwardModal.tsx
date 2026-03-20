import { Modal, Form, Input, InputNumber, Select, message } from 'antd'
import { addPortForward } from '../../api/client'

interface Props {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function AddPortForwardModal({ open, onClose, onSuccess }: Props) {
  const [form] = Form.useForm()

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      await addPortForward(values)
      message.success('Port forward added')
      form.resetFields()
      onClose()
      onSuccess()
    } catch {
      message.error('Failed to add port forward')
    }
  }

  return (
    <Modal title="Add Port Forward" open={open} onOk={handleOk} onCancel={onClose} destroyOnClose>
      <Form form={form} layout="vertical" initialValues={{ protocol: 'tcp' }}>
        <Form.Item name="name" label="Name" rules={[{ required: true }]}>
          <Input placeholder="e.g. web-server" />
        </Form.Item>
        <Form.Item name="protocol" label="Protocol" rules={[{ required: true }]}>
          <Select options={[{ value: 'tcp', label: 'TCP' }, { value: 'udp', label: 'UDP' }]} />
        </Form.Item>
        <Form.Item name="external_port" label="External Port" rules={[{ required: true }]}>
          <InputNumber min={1} max={65535} className="w-full" />
        </Form.Item>
        <Form.Item
          name="internal_ip"
          label="Internal IP"
          rules={[
            { required: true },
            { pattern: /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/, message: 'Invalid IP' },
          ]}
        >
          <Input placeholder="192.168.1.100" />
        </Form.Item>
        <Form.Item name="internal_port" label="Internal Port" rules={[{ required: true }]}>
          <InputNumber min={1} max={65535} className="w-full" />
        </Form.Item>
      </Form>
    </Modal>
  )
}
