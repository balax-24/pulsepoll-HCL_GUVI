import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext.jsx'
import { Navbar } from './components/Navbar.jsx'
import { Footer } from './components/Footer.jsx'
import { ProtectedRoute } from './components/ProtectedRoute.jsx'

// Pages
import { HomePage } from './pages/HomePage.jsx'
import { LoginPage } from './pages/LoginPage.jsx'
import { RegisterPage } from './pages/RegisterPage.jsx'
import { DashboardPage } from './pages/DashboardPage.jsx'
import { CreatePollPage } from './pages/CreatePollPage.jsx'
import { PublicPollPage } from './pages/PublicPollPage.jsx'
import { ManagePollPage } from './pages/ManagePollPage.jsx'

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <div className="app-shell">
          <Navbar />
          <main className="main-content">
            <Routes>
              {/* Public Routes */}
              <Route path="/" element={<HomePage />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />

              {/* Public Poll Access (No Authentication Guard) */}
              <Route path="/polls/:id" element={<PublicPollPage />} />

              {/* Protected Creator Routes */}
              <Route
                path="/dashboard"
                element={
                  <ProtectedRoute>
                    <DashboardPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/polls/create"
                element={
                  <ProtectedRoute>
                    <CreatePollPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/polls/:id/manage"
                element={
                  <ProtectedRoute>
                    <ManagePollPage />
                  </ProtectedRoute>
                }
              />

              {/* Fallback */}
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </main>
          <Footer />
        </div>
      </BrowserRouter>
    </AuthProvider>
  )
}

export default App
