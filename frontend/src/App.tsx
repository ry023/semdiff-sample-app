import { createContext, useCallback, useContext, useEffect, useMemo, useState, type FormEvent, type ReactNode } from 'react'
import {
  Alert,
  AppBar,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  CircularProgress,
  Container,
  Divider,
  IconButton,
  Link as MuiLink,
  Paper,
  Stack,
  TextField,
  Toolbar,
  Typography,
} from '@mui/material'
import AddRoundedIcon from '@mui/icons-material/AddRounded'
import DeleteOutlineRoundedIcon from '@mui/icons-material/DeleteOutlineRounded'
import ExitToAppRoundedIcon from '@mui/icons-material/ExitToAppRounded'
import FormatListBulletedRoundedIcon from '@mui/icons-material/FormatListBulletedRounded'
import { Link, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { api } from './api'
import type { Todo, User } from './types'

type AuthContextValue = {
  user: User | null
  loading: boolean
  login: (user: User) => void
  logout: () => Promise<void>
}

const AuthContext = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.me()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      loading,
      login: setUser,
      logout: async () => {
        await api.logout()
        setUser(null)
      },
    }),
    [loading, user],
  )

  return <AuthContextProvider value={value}>{children}</AuthContextProvider>
}

const authContext = createContext<AuthContextValue | null>(null)
const AuthContextProvider = authContext.Provider

function useAuth() {
  const value = useContext(authContext)
  if (!value) {
    throw new Error('useAuth must be used inside AuthContext')
  }
  return value
}

function App() {
  return (
    <AuthContext>
      <Routes>
        <Route path="/" element={<Navigate to="/app" replace />} />
        <Route path="/login" element={<AuthPage mode="login" />} />
        <Route path="/register" element={<AuthPage mode="register" />} />
        <Route path="/app" element={<ProtectedRoute><TodoPage /></ProtectedRoute>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AuthContext>
  )
}

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return <LoadingScreen />
  }
  if (!user) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }
  return <>{children}</>
}

function LoadingScreen() {
  return (
    <Box minHeight="100vh" display="grid" sx={{ placeItems: 'center' }}>
      <CircularProgress />
    </Box>
  )
}

function AuthPage({ mode }: { mode: 'login' | 'register' }) {
  const { user, login } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const isRegister = mode === 'register'

  if (user) {
    return <Navigate to="/app" replace />
  }

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      if (isRegister) {
        await api.register(username, password)
      }
      const loggedIn = await api.login(username, password)
      login(loggedIn)
      navigate('/app', { replace: true })
    } catch (submissionError) {
      setError(submissionError instanceof Error ? submissionError.message : '処理に失敗しました')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Box sx={{ minHeight: '100vh', display: 'grid', placeItems: 'center', p: 2 }}>
      <Container maxWidth="sm">
        <Card>
          <CardContent sx={{ p: { xs: 3, sm: 5 } }}>
            <Stack spacing={3}>
              <Box>
                <Stack direction="row" spacing={1} alignItems="center" mb={2}>
                  <FormatListBulletedRoundedIcon color="secondary" />
                  <Typography color="primary" fontWeight={800} letterSpacing="0.08em">
                    TODO GARDEN
                  </Typography>
                </Stack>
                <Typography variant="h2" component="h1" gutterBottom>
                  {isRegister ? '今日のタスクを育てよう' : 'おかえりなさい'}
                </Typography>
                <Typography color="text.secondary">
                  {isRegister ? 'アカウントを作って、やることを整理します。' : 'タスクの続きを始めましょう。'}
                </Typography>
              </Box>

              {error && <Alert severity="error">{error}</Alert>}

              <Box component="form" onSubmit={submit}>
                <Stack spacing={2}>
                  <TextField
                    label="ユーザー名"
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                    autoComplete="username"
                    required
                    fullWidth
                  />
                  <TextField
                    label="パスワード"
                    type="password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    autoComplete={isRegister ? 'new-password' : 'current-password'}
                    required
                    fullWidth
                  />
                  <Button type="submit" variant="contained" size="large" disabled={submitting}>
                    {submitting ? <CircularProgress size={24} color="inherit" /> : isRegister ? '登録して始める' : 'ログイン'}
                  </Button>
                </Stack>
              </Box>

              <Divider />
              <Typography textAlign="center" color="text.secondary">
                {isRegister ? 'すでにアカウントをお持ちですか？ ' : 'はじめて利用しますか？ '}
                <MuiLink component={Link} to={isRegister ? '/login' : '/register'} underline="hover">
                  {isRegister ? 'ログイン' : '新規登録'}
                </MuiLink>
              </Typography>
            </Stack>
          </CardContent>
        </Card>
      </Container>
    </Box>
  )
}

function TodoPage() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [todos, setTodos] = useState<Todo[]>([])
  const [title, setTitle] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  const loadTodos = useCallback(async () => {
    try {
      setTodos(await api.listTodos())
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Todoを読み込めませんでした')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadTodos()
  }, [loadTodos])

  const addTodo = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!title.trim()) return
    setError('')
    try {
      const created = await api.createTodo(title.trim())
      setTodos((current) => [created, ...current])
      setTitle('')
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : 'Todoを追加できませんでした')
    }
  }

  const toggleTodo = async (todo: Todo) => {
    try {
      const updated = await api.updateTodo(todo.id, { completed: !todo.completed })
      setTodos((current) => current.map((item) => (item.id === updated.id ? updated : item)))
    } catch (updateError) {
      setError(updateError instanceof Error ? updateError.message : 'Todoを更新できませんでした')
    }
  }

  const deleteTodo = async (todo: Todo) => {
    try {
      await api.deleteTodo(todo.id)
      setTodos((current) => current.filter((item) => item.id !== todo.id))
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : 'Todoを削除できませんでした')
    }
  }

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  const remaining = todos.filter((todo) => !todo.completed).length

  return (
    <Box minHeight="100vh">
      <AppBar position="static" color="transparent" elevation={0}>
        <Toolbar sx={{ maxWidth: 1100, width: '100%', mx: 'auto' }}>
          <Stack direction="row" spacing={1} alignItems="center" flexGrow={1}>
            <FormatListBulletedRoundedIcon color="secondary" />
            <Typography color="primary" fontWeight={800} letterSpacing="0.08em">
              TODO GARDEN
            </Typography>
          </Stack>
          <Typography variant="body2" color="text.secondary" mr={2} display={{ xs: 'none', sm: 'block' }}>
            {user?.username}
          </Typography>
          <Button color="primary" startIcon={<ExitToAppRoundedIcon />} onClick={handleLogout}>
            ログアウト
          </Button>
        </Toolbar>
      </AppBar>

      <Container maxWidth="md" sx={{ py: { xs: 4, sm: 8 } }}>
        <Stack spacing={4}>
          <Box>
            <Typography variant="h1" component="h1" sx={{ fontSize: { xs: '2.5rem', sm: '4rem' } }}>
              やることを、<br />
              ひとつずつ。
            </Typography>
            <Typography color="text.secondary" mt={2} fontSize="1.1rem">
              {remaining === 0 ? 'すべて完了しました。いい流れです。' : `未完了のタスクが ${remaining} 件あります。`}
            </Typography>
          </Box>

          <Paper component="form" onSubmit={addTodo} sx={{ p: 1, pl: 2, borderRadius: 4, display: 'flex', gap: 1 }}>
            <TextField
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="新しいTodoを追加"
              variant="standard"
              fullWidth
              InputProps={{ disableUnderline: true }}
              inputProps={{ 'aria-label': '新しいTodo' }}
            />
            <Button type="submit" variant="contained" startIcon={<AddRoundedIcon />} sx={{ px: 2.5, flexShrink: 0 }}>
              追加
            </Button>
          </Paper>

          {error && <Alert severity="error">{error}</Alert>}

          <Card>
            <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
              {loading ? (
                <Box p={5} display="grid" sx={{ placeItems: 'center' }}><CircularProgress /></Box>
              ) : todos.length === 0 ? (
                <Box p={{ xs: 4, sm: 7 }} textAlign="center">
                  <Typography variant="h6" color="primary" gutterBottom>まだTodoがありません</Typography>
                  <Typography color="text.secondary">頭の中のタスクを、まずひとつ書き出してみましょう。</Typography>
                </Box>
              ) : (
                todos.map((todo, index) => (
                  <Box key={todo.id}>
                    <Stack direction="row" alignItems="center" spacing={1.5} px={{ xs: 2, sm: 3 }} py={1.5}>
                      <Checkbox checked={todo.completed} onChange={() => void toggleTodo(todo)} color="secondary" />
                      <Typography
                        flexGrow={1}
                        sx={{ textDecoration: todo.completed ? 'line-through' : 'none', color: todo.completed ? 'text.disabled' : 'text.primary' }}
                      >
                        {todo.title}
                      </Typography>
                      <IconButton aria-label={`${todo.title}を削除`} onClick={() => void deleteTodo(todo)} color="default">
                        <DeleteOutlineRoundedIcon />
                      </IconButton>
                    </Stack>
                    {index < todos.length - 1 && <Divider />}
                  </Box>
                ))
              )}
            </CardContent>
          </Card>
        </Stack>
      </Container>
    </Box>
  )
}

export default App
