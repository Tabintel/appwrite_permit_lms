"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/router"
import { useAuth } from "../contexts/auth-context"
import { type Course, getCourses } from "../lib/appwrite"
import Link from "next/link"

export default function Dashboard() {
  const { user, isLoading: authLoading, logout } = useAuth()
  const router = useRouter()
  const [courses, setCourses] = useState<Course[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState("")

  useEffect(() => {
    if (!authLoading && !user) {
      router.push("/login")
    }
  }, [user, authLoading, router])

  useEffect(() => {
    const fetchCourses = async () => {
      if (user && user.$id && user.role) {
        try {
          const fetchedCourses = await getCourses(user.$id, user.role)
          setCourses(fetchedCourses)
        } catch (error) {
          console.error("Error fetching courses:", error)
          setError("Failed to load courses")
        } finally {
          setIsLoading(false)
        }
      }
    }

    if (user) {
      fetchCourses()
    }
  }, [user])

  const handleLogout = async () => {
    try {
      await logout()
      router.push("/")
    } catch (error) {
      console.error("Logout error:", error)
    }
  }

  if (authLoading || !user) {
    return <div className="min-h-screen bg-gray-100 flex items-center justify-center">Loading...</div>
  }

  return (
    <div className="min-h-screen bg-gray-100">
      <header className="bg-white shadow">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-2xl font-bold">LMS Dashboard</h1>
          <div className="flex items-center space-x-4">
            <div className="text-sm">
              <span className="block text-gray-500">Logged in as</span>
              <span className="font-medium">
                {user.name} ({user.role})
              </span>
            </div>
            <button onClick={handleLogout} className="px-3 py-1 bg-red-500 text-white rounded hover:bg-red-600">
              Logout
            </button>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <h2 className="text-xl font-semibold">Your Courses</h2>
          {(user.role === "teacher" || user.role === "admin") && (
            <Link href="/courses/create" className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Create Course</Link>
          )}
        </div>

        {isLoading ? (
          <div className="text-center py-8">Loading courses...</div>
        ) : error ? (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">{error}</div>
        ) : courses.length === 0 ? (
          <div className="bg-white p-8 rounded-lg shadow-md text-center">
            <p className="text-gray-600 mb-4">You don't have any courses yet.</p>
            {user.role === "student" && (
              <Link href="/courses" className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Browse Courses</Link>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {courses.map((course) => (
              <Link href={`/courses/${course.$id}`} key={course.$id} className="block bg-white p-6 rounded-lg shadow-md hover:shadow-lg transition-shadow">
                <h3 className="text-lg font-semibold mb-2">{course.title}</h3>
                <p className="text-gray-600 mb-4">{course.description}</p>
                <div className="text-sm text-gray-500">
                  {user.role === "teacher" ? (
                    <span>Students: {course.studentIds.length}</span>
                  ) : user.role === "student" ? (
                    <span>Teacher: {course.teacherId}</span>
                  ) : (
                    <div className="flex justify-between">
                      <span>Teacher: {course.teacherId}</span>
                      <span>Students: {course.studentIds.length}</span>
                    </div>
                  )}
                </div>
              </Link>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
