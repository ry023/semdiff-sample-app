import type { Todo, User } from './types'

type ErrorResponse = {
  error?: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ErrorResponse
    throw new Error(body.error ?? '通信に失敗しました')
  }

  if (response.status === 204) {
    return undefined as T
  }
  return (await response.json()) as T
}

export const api = {
  register: (username: string, password: string) =>
    request<User>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  login: (username: string, password: string) =>
    request<User>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request<void>('/api/auth/logout', { method: 'POST' }),
  me: () => request<User>('/api/auth/me'),
  listTodos: () => request<Todo[]>('/api/todos'),
  createTodo: (title: string) =>
    request<Todo>('/api/todos', {
      method: 'POST',
      body: JSON.stringify({ title }),
    }),
  updateTodo: (id: number, changes: Partial<Pick<Todo, 'title' | 'completed'>>) =>
    request<Todo>(`/api/todos/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(changes),
    }),
  deleteTodo: (id: number) => request<void>(`/api/todos/${id}`, { method: 'DELETE' }),
}

