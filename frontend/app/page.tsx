import Link from "next/link"

export default function Home() {
  return (
    <div className="min-h-screen bg-gray-100">
      <div className="container mx-auto px-4 py-8">
        <header className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold">LMS App</h1>
          <div className="space-x-4">
            <Link href="/login" className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Login</Link>
            <Link href="/register" className="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600">Register</Link>
          </div>
        </header>

        <main>
          <div className="bg-white p-8 rounded-lg shadow-md">
            <h2 className="text-2xl font-bold mb-4">Welcome to the Learning Management System</h2>
            <p className="mb-4">
              This is a demo application showcasing fine-grained authorization with Appwrite and Permit.io.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-8">
              <div className="bg-blue-50 p-6 rounded-lg">
                <h3 className="text-xl font-semibold mb-2">For Students</h3>
                <p>Enroll in courses, view assignments, and submit your work.</p>
              </div>
              <div className="bg-green-50 p-6 rounded-lg">
                <h3 className="text-xl font-semibold mb-2">For Teachers</h3>
                <p>Create courses, manage assignments, and grade student submissions.</p>
              </div>
              <div className="bg-purple-50 p-6 rounded-lg">
                <h3 className="text-xl font-semibold mb-2">For Admins</h3>
                <p>Manage all courses, assignments, and users in the system.</p>
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
