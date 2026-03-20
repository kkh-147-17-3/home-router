import { Modal, Form, Input, message } from 'antd'
import { addStaticLease } from '../../api/client'

interface Props {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function AddStaticLeaseModal({ open, onClose, onSuccess }: Props) {
  const [form] = Form.useForm()

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      await addStaticLease(values)
      message.success('Static lease added')
      form.resetFields()
      onClose()
      onSuccess()
    } catch {
      message.error('Failed to add static lease')
    }
  }

  return (
    <Modal title="Add Static Lease" open={open} onOk={handleOk} onCancel={onClose} destroyOnClose>
      <Form form={form} layout="vertical">
        <Form.Item name="name" label="Name" rules={[{ required: true }]}>
          <Input placeholder="e.g. My-Device" />
        </Form.Item>
        <Form.Item
          name="mac"
          label="MAC Address"
          rules={[
            { required: true },
            { pattern: /^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$/, message: 'Invalid MAC (xx:xx:xx:xx:xx:xx)' },
          ]}
        >
          <Input placeholder="00:11:22:33:44:55" />
        </Form.Item>
        <Form.Item
          name="ip"
          label="IP Address"
          rules={[
            { required: true },
            { pattern: /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/, message: 'Invalid IP' },
          ]}
        >
          <Input placeholder="192.168.1.100" />
        </Form.Item>
      </Form>
    </Modal>
  )
}
