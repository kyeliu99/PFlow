import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FormEvent, useState } from 'react';
import { approveTicket, createTicket, fetchTickets, submitTicket, Ticket } from '../api';
import { TicketCard } from '../components/TicketCard';

export function App() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState({ title: '', description: '', requester: '', assignee: '' });

  const { data: tickets = [], isLoading } = useQuery({ queryKey: ['tickets'], queryFn: fetchTickets, refetchInterval: 5000 });

  const createMutation = useMutation({
    mutationFn: createTicket,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      setForm({ title: '', description: '', requester: '', assignee: '' });
    }
  });

  const submitMutation = useMutation({
    mutationFn: submitTicket,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tickets'] })
  });

  const decisionMutation = useMutation({
    mutationFn: ({ id, approved }: { id: string; approved: boolean }) => approveTicket(id, approved),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tickets'] })
  });

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    createMutation.mutate(form);
  };

  return (
    <div className="container">
      <header>
        <h1>PFlow Tickets</h1>
        <p>Camunda 驱动的解耦工作流示例</p>
      </header>

      <section className="form">
        <h2>创建工单</h2>
        <form onSubmit={handleSubmit}>
          <label>
            标题
            <input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required />
          </label>
          <label>
            描述
            <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </label>
          <label>
            申请人
            <input value={form.requester} onChange={(e) => setForm({ ...form, requester: e.target.value })} required />
          </label>
          <label>
            指派给
            <input value={form.assignee} onChange={(e) => setForm({ ...form, assignee: e.target.value })} />
          </label>
          <button type="submit" disabled={createMutation.isLoading}>创建</button>
        </form>
      </section>

      <section>
        <h2>工单列表</h2>
        {isLoading ? <p>加载中...</p> : null}
        <div className="ticket-grid">
          {tickets.map((ticket: Ticket) => (
            <TicketCard
              key={ticket.id}
              ticket={ticket}
              onSubmit={() => submitMutation.mutate(ticket.id)}
              onApprove={(approved) => decisionMutation.mutate({ id: ticket.id, approved })}
              isSubmitting={submitMutation.isLoading}
              isDeciding={decisionMutation.isLoading}
            />
          ))}
        </div>
      </section>
    </div>
  );
}
