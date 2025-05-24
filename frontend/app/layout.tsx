import { AuthProvider } from '../contexts/auth-context'
import './globals.css'

export const metadata = {
  title: 'LMS App',
  description: 'Learning Management System with Appwrite and Permit.io',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen bg-background antialiased">
        <AuthProvider>
          {children}
        </AuthProvider>
      </body>
    </html>
  )
}
