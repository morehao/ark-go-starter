import React from 'react'
import ReactDOM from 'react-dom/client'
import { AppShell } from '@ark-go-starter/ui'
import App from './App'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppShell>
      <App />
    </AppShell>
  </React.StrictMode>,
)
