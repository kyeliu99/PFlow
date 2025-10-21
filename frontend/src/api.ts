import axios from 'axios';

export interface Ticket {
  id: string;
  title: string;
  description: string;
  requester: string;
  assignee: string;
  status: string;
  processInstanceId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateTicketInput {
  title: string;
  description: string;
  requester: string;
  assignee?: string;
}

export async function fetchTickets(): Promise<Ticket[]> {
  const { data } = await axios.get<Ticket[]>('/api/tickets');
  return data;
}

export async function createTicket(input: CreateTicketInput): Promise<Ticket> {
  const { data } = await axios.post<Ticket>('/api/tickets', input);
  return data;
}

export async function submitTicket(id: string): Promise<void> {
  await axios.post(`/api/tickets/${id}/submit`);
}

export async function approveTicket(id: string, approved: boolean, comment?: string): Promise<void> {
  await axios.post(`/api/tickets/${id}/decision`, { approved, comment });
}
