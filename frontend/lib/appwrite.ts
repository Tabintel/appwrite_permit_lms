import { Account, Client, Databases, ID, type Models, Storage } from "appwrite"

// Initialize Appwrite client
const client = new Client()

client
  .setEndpoint(process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT || "https://cloud.appwrite.io/v1")
  .setProject(process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID || "")

// Initialize Appwrite services
export const account = new Account(client)
export const databases = new Databases(client)
export const storage = new Storage(client)

// Database and collection IDs
export const DATABASE_ID = process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID || ""
export const COURSES_COLLECTION_ID = "courses"
export const ASSIGNMENTS_COLLECTION_ID = "assignments"
export const SUBMISSIONS_COLLECTION_ID = "submissions"

// Types
export interface User extends Models.User {
  role?: "admin" | "teacher" | "student"
}

export interface Course {
  $id?: string
  title: string
  description: string
  teacherId: string
  studentIds: string[]
}

export interface Assignment {
  $id?: string
  title: string
  description: string
  courseId: string
  dueDate: string
}

export interface Submission {
  $id?: string
  assignmentId: string
  studentId: string
  content: string
  submittedAt: string
  grade: number
  feedback: string
}

// Auth functions
export const createAccount = async (
  email: string,
  password: string,
  name: string,
  role: "admin" | "teacher" | "student",
): Promise<User> => {
  try {
    // Create account
    const user = await account.create(ID.unique(), email, password, name)

    // Update user preferences to store role
    await account.updatePrefs({ role })

    return { ...user, role }
  } catch (error) {
    console.error("Error creating account:", error)
    throw error
  }
}

export const login = async (email: string, password: string): Promise<User> => {
  try {
    // Create email session
    await account.createEmailSession(email, password)

    // Get account
    const user = await account.get()

    // Get user preferences
    const prefs = await account.getPrefs()

    return { ...user, role: prefs.role }
  } catch (error) {
    console.error("Error logging in:", error)
    throw error
  }
}

export const logout = async (): Promise<void> => {
  try {
    await account.deleteSession("current")
  } catch (error) {
    console.error("Error logging out:", error)
    throw error
  }
}

export const getCurrentUser = async (): Promise<User | null> => {
  try {
    const user = await account.get()
    const prefs = await account.getPrefs()

    return { ...user, role: prefs.role }
  } catch (error) {
    return null
  }
}

// Course functions
export const getCourses = async (userId: string, userRole: string): Promise<Course[]> => {
  try {
    // Call Appwrite function to get courses
    const response = await client.functions.createExecution("get_courses", JSON.stringify({ userId, userRole }), false)

    const result = JSON.parse(response.response)

    if (!result.success) {
      throw new Error(result.message)
    }

    return result.data
  } catch (error) {
    console.error("Error getting courses:", error)
    throw error
  }
}

export const createCourse = async (
  userId: string,
  userRole: string,
  title: string,
  description: string,
): Promise<Course> => {
  try {
    // Call Appwrite function to create course
    const response = await client.functions.createExecution(
      "create_course",
      JSON.stringify({ userId, userRole, title, description }),
      false,
    )

    const result = JSON.parse(response.response)

    if (!result.success) {
      throw new Error(result.message)
    }

    return result.data
  } catch (error) {
    console.error("Error creating course:", error)
    throw error
  }
}

export const enrollInCourse = async (userId: string, userRole: string, courseId: string): Promise<Course> => {
  try {
    // Call Appwrite function to enroll in course
    const response = await client.functions.createExecution(
      "enroll_course",
      JSON.stringify({ userId, userRole, courseId }),
      false,
    )

    const result = JSON.parse(response.response)

    if (!result.success) {
      throw new Error(result.message)
    }

    return result.data
  } catch (error) {
    console.error("Error enrolling in course:", error)
    throw error
  }
}

// Assignment functions
export const getAssignments = async (userId: string, userRole: string, courseId: string): Promise<Assignment[]> => {
  try {
    // Call Appwrite function to get assignments
    const response = await client.functions.createExecution(
      "get_assignments",
      JSON.stringify({ userId, userRole, courseId }),
      false,
    )

    const result = JSON.parse(response.response)

    if (!result.success) {
      throw new Error(result.message)
    }

    return result.data
  } catch (error) {
    console.error("Error getting assignments:", error)
    throw error
  }
}

export const submitAssignment = async (
  userId: string,
  userRole: string,
  assignmentId: string,
  content: string,
): Promise<Submission> => {
  try {
    // Call Appwrite function to submit assignment
    const response = await client.functions.createExecution(
      "submit_assignment",
      JSON.stringify({ userId, userRole, assignmentId, content }),
      false,
    )

    const result = JSON.parse(response.response)

    if (!result.success) {
      throw new Error(result.message)
    }

    return result.data
  } catch (error) {
    console.error("Error submitting assignment:", error)
    throw error
  }
}

export const gradeAssignment = async (
  userId: string,
  userRole: string,
  submissionId: string,
  grade: number,
  feedback: string,
): Promise<Submission> => {
  try {
    // Call Appwrite function to grade assignment
    const response = await client.functions.createExecution(
      "grade_assignment",
      JSON.stringify({ userId, userRole, submissionId, grade, feedback }),
      false,
    )

    const result = JSON.parse(response.response)

    if (!result.success) {
      throw new Error(result.message)
    }

    return result.data
  } catch (error) {
    console.error("Error grading assignment:", error)
    throw error
  }
}
