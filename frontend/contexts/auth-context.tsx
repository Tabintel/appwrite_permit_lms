"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"
import {
  type User,
  getCurrentUser,
  login as appwriteLogin,
  logout as appwriteLogout,
  createAccount as appwriteCreateAccount,
} from "../lib/appwrite"

interface AuthContextType {
  user: User | null
  isLoading: boolean
  login: (email: string, password: string) => Promise<User>
  logout: () => Promise<void>
  createAccount: (email: string, password: string, name: string, role: "admin" | "teacher" | "student") => Promise<User>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const checkUser = async () => {
      try {
        const currentUser = await getCurrentUser()
        setUser(currentUser)
      } catch (error) {
        console.error("Error checking user:", error)
      } finally {
        setIsLoading(false)
      }
    }

    checkUser()
  }, [])

  const login = async (email: string, password: string) => {
    try {
      const user = await appwriteLogin(email, password)
      setUser(user)
      return user
    } catch (error) {
      console.error("Error logging in:", error)
      throw error
    }
  }

  const logout = async () => {
    try {
      await appwriteLogout()
      setUser(null)
    } catch (error) {
      console.error("Error logging out:", error)
      throw error
    }
  }

  const createAccount = async (
    email: string,
    password: string,
    name: string,
    role: "admin" | "teacher" | "student",
  ) => {
    try {
      const user = await appwriteCreateAccount(email, password, name, role)
      setUser(user)
      return user
    } catch (error) {
      console.error("Error creating account:", error)
      throw error
    }
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout, createAccount }}>{children}</AuthContext.Provider>
  )
}

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}
