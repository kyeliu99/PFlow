import { Ticket } from '../api';

interface Props {
  ticket: Ticket;
  onSubmit: () => void;
  onApprove: (approved: boolean) => void;
  isSubmitting: boolean;
  isDeciding: boolean;
}

const statusLabel: Record<string, string> = {
  draft: '草稿',
  submitted: '待审批',
  approved: '已通过',
  rejected: '已驳回',
  processing: '处理中',
  completed: '已完成'
};

export function TicketCard({ ticket, onSubmit, onApprove, isSubmitting, isDeciding }: Props) {
  const showSubmit = ticket.status === 'draft' || ticket.status === 'rejected';
  const showDecision = ticket.status === 'submitted';

  return (
    <article className="ticket-card">
      <h3>{ticket.title}</h3>
      <p className="meta">工单号：{ticket.id}</p>
      <p>{ticket.description || '无描述'}</p>
      <p className="meta">申请人：{ticket.requester}</p>
      <p className="meta">当前状态：{statusLabel[ticket.status] ?? ticket.status}</p>
      <div className="actions">
        {showSubmit && (
          <button onClick={onSubmit} disabled={isSubmitting}>
            {isSubmitting ? '提交中...' : '提交审批'}
          </button>
        )}
        {showDecision && (
          <>
            <button onClick={() => onApprove(true)} disabled={isDeciding}>
              {isDeciding ? '处理中...' : '通过'}
            </button>
            <button onClick={() => onApprove(false)} disabled={isDeciding} className="danger">
              {isDeciding ? '处理中...' : '驳回'}
            </button>
          </>
        )}
      </div>
    </article>
  );
}
