// 二次确认弹窗，用于删除、验收合格等不可逆操作。
import type { ReactNode } from 'react';
import { Modal } from './Modal';

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: ReactNode;
  confirmText?: string;
  danger?: boolean;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmText = '确认',
  danger,
  busy,
  onConfirm,
  onCancel
}: ConfirmDialogProps) {
  return (
    <Modal
      open={open}
      title={title}
      onClose={onCancel}
      width={460}
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onCancel} disabled={busy}>
            取消
          </button>
          <button
            type="button"
            className={`btn ${danger ? 'btn-danger' : 'btn-primary'}`}
            onClick={onConfirm}
            disabled={busy}
          >
            {busy ? '处理中…' : confirmText}
          </button>
        </>
      }
    >
      <div className="confirm-body">{message}</div>
    </Modal>
  );
}
