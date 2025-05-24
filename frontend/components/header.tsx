"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { Button } from "@/components/ui/button"
import { ModeToggle } from "@/components/mode-toggle"
import { LogOut, User, BookOpen, FileText, Home } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useAuth } from "@/contexts/auth-context"

export function Header() {
  const pathname = usePathname()
  const { user, logout } = useAuth()

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-16 items-center justify-between">
        <div className="flex items-center gap-6 md:gap-10">
          <Link href="/" className="flex items-center space-x-2">
            <div className="h-6 w-6 bg-gradient-to-r from-primary to-accent rounded-md"></div>
            <span className="font-bold">LMS App</span>
          </Link>
          <nav className="hidden gap-6 md:flex">
            <Link href="/dashboard" className={`text-sm font-medium transition-colors hover:text-primary flex items-center gap-1 ${pathname === "/dashboard" ? "text-primary" : "text-foreground/60"}`}>
              <Home className="mr-2 h-4 w-4" />
              Dashboard
            </Link>
            <Link href="/courses" className={`text-sm font-medium transition-colors hover:text-primary flex items-center gap-1 ${pathname.startsWith("/courses") ? "text-primary" : "text-foreground/60"}`}>
              <BookOpen className="mr-2 h-4 w-4" />
              Courses
            </Link>
            <Link
              href="/assignments"
              className={`text-sm font-medium transition-colors hover:text-primary flex items-center gap-1 ${
                pathname.startsWith("/assignments") ? "text-primary" : "text-foreground/60"
              }`}
            >
              <FileText className="h-4 w-4" />
              Assignments
            </Link>
          </nav>
        </div>
        <div className="flex items-center gap-2">
          <ModeToggle />
          {user ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="relative">
                  <User className="h-5 w-5" />
                  <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground">
                    {user.role.charAt(0).toUpperCase()}
                  </span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>
                  <div className="flex flex-col space-y-1">
                    <p className="text-sm font-medium leading-none">{user.name}</p>
                    <p className="text-xs leading-none text-muted-foreground">{user.email}</p>
                    <p className="text-xs leading-none text-muted-foreground capitalize">Role: {user.role}</p>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={logout}>
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>Log out</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <Button asChild variant="default" size="sm">
              <Link href="/login" className="block">Login</Link>
            </Button>
          )}
        </div>
      </div>
    </header>
  )
}
