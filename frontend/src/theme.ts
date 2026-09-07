import { createTheme } from '@mui/material/styles'

export const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#153b50',
    },
    secondary: {
      main: '#d17b49',
    },
    background: {
      default: '#f6f3ed',
      paper: '#fffdf8',
    },
  },
  typography: {
    fontFamily: '"Avenir Next", "Hiragino Sans", "Yu Gothic", sans-serif',
    h1: {
      fontWeight: 800,
      letterSpacing: '-0.04em',
    },
    h2: {
      fontWeight: 800,
      letterSpacing: '-0.03em',
    },
    button: {
      fontWeight: 700,
      textTransform: 'none',
    },
  },
  shape: {
    borderRadius: 16,
  },
  components: {
    MuiButton: {
      defaultProps: {
        disableElevation: true,
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          border: '1px solid rgba(21, 59, 80, 0.08)',
          boxShadow: '0 16px 50px rgba(21, 59, 80, 0.08)',
        },
      },
    },
  },
})

