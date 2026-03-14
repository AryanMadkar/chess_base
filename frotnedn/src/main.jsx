import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'

// StrictMode removed — it calls functions twice in dev, causing onPieceDrop
// to fire twice and sending duplicate moves to the backend.
createRoot(document.getElementById('root')).render(<App />)