"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/router"
import { useAuth } from "../../contexts/auth-context"
import { type Course, getCourses, enrollInCourse } from "../../lib/appwrite"
import Link from "next/link"

export default function CoursesPage() {
  const { user, isLoading: authLoading } = useAuth()
  const router = useRouter()
  const [courses, setCourses] = useState<Course[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState("")
  const [enrollingCourseId, setEnrollingCourseId] = useState<string | null>(null)

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

  const handleEnroll = async (courseId: string) => {
    if (!user || !user.$id || !user.role) return

    setEnrollingCourseId(courseId)
    try {
      await enrollInCourse(user.$id, user.role, courseId)

      // Update the course in the local state
      setCourses(
        courses.map((course) => {
          if (course.$id === courseId) {
            return {
              ...course,
              studentIds: [...course.studentIds, user.$id],
            }
          }
          return course
        }),
      )
    } catch (error) {
      console.error("Error enrolling in course:", error)
      setError("Failed to enroll in course")
    } finally {
      setEnrollingCourseId(null)
    }
  }

  const isEnrolled = (course: Course) => {
    return user && course.studentIds.includes(user.$id)
  }

  const isTeacher = (course: Course) => {
    return user && course.teacherId === user.$id
  }

  if (authLoading || !user) {
    return <div className="min-h-screen bg-gray-100 flex items-center justify-center">Loading...</div>
  }

  return (
    <div className="min-h-screen bg-gray-100">
      <header className="bg-white shadow">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-2xl font-bold">Courses</h1>
          <div className="flex items-center space-x-4">
            <Link href="/dashboard" className="px-3 py-1 bg-gray-200 text-gray-700 rounded hover:bg-gray-300">Dashboard</Link>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <h2 className="text-xl font-semibold">Available Courses</h2>
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
            <p className="text-gray-600">No courses available.</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {courses.map((course) => (
              <div key={course.$id} className="bg-white rounded-lg shadow-md overflow-hidden">
                <div className="p-6">
                  <h3 className="text-lg font-semibold mb-2">{course.title}</h3>
                  <p className="text-gray-600 mb-4">{course.description}</p>
                  <div className="text-sm text-gray-500 mb-4">
                    <p>Teacher: {course.teacherId === user.$id ? "You" : course.teacherId}</p>
                    <p>Students: {course.studentIds.length}</p>
                  </div>
                  <div className="flex justify-between items-center">
                    <Link href={`/courses/${course.$id}`} className="text-blue-500 hover:text-blue-700">View Details</Link>
                    {user.role === "student" && !isEnrolled(course) && !isTeacher(course) && (
                      <button
                        onClick={() => handleEnroll(course.$id)}
                        disabled={enrollingCourseId === course.$id}
                        className="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600 disabled:opacity-50"
                      >
                        {enrollingCourseId === course.$id ? "Enrolling..." : "Enroll"}
                      </button>
                    )}
                    {isEnrolled(course) && (
                      <span className="px-4 py-2 bg-gray-200 text-gray-700 rounded">Enrolled</span>
                    )}
                    {isTeacher(course) && (
                      <span className="px-4 py-2 bg-blue-200 text-blue-700 rounded">Your Course</span>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
