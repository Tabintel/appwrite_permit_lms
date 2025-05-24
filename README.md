# LMS Application with Appwrite and Permit.io

This project is a Learning Management System (LMS) that demonstrates how to implement fine-grained authorization using Appwrite for authentication and database, Permit.io for authorization policies, and Go for backend logic, with a Next.js frontend.

## Architecture

The application follows this architecture:

1. **Backend (Go)**: Handles business logic as Appwrite Cloud Functions.
2. **Appwrite**: Provides authentication and database services.
3. **Permit.io**: Manages fine-grained authorization policies.
4. **Frontend (Next.js)**: Provides the user interface for students, teachers, and admins.

## Features

- **User Authentication**: Sign up and login with different roles (student, teacher, admin).
- **Role-Based Access Control**: Different UI and permissions based on user roles.
- **Fine-Grained Authorization**: Access control based on user attributes and resource relationships.
- **Course Management**: Create, view, update, and delete courses based on permissions.
- **Assignment Management**: Create, submit, and grade assignments based on permissions.

## Project Structure

- `backend/`: Go functions deployed as Appwrite Cloud Functions
- `frontend/`: Next.js frontend connecting to Appwrite + Permit
- Appwrite manages all user data and database collections
- Permit holds and evaluates dynamic access control rules

## Setup

### Prerequisites

- Node.js 14+
- Go 1.16+
- Appwrite instance
- Permit.io account


### Deploying the Backend Functions

1. Install the Appwrite CLI
2. Login to your Appwrite instance
3. Deploy the functions:

```bash
cd backend/functions
appwrite functions create get_courses --runtime go-1.19 --entrypoint main
appwrite functions create create_course --runtime go-1.19 --entrypoint main
appwrite functions create enroll_course --runtime go-1.19 --entrypoint main
appwrite functions create get_assignments --runtime go-1.19 --entrypoint main
appwrite functions create submit_assignment --runtime go-1.19 --entrypoint main
appwrite functions create grade_assignment --runtime go-1.19 --entrypoint main
```

4. Set environment variables for each function:

```bash
appwrite functions createVariable get_courses APPWRITE_API_KEY "your-api-key"
appwrite functions createVariable get_courses PERMIT_TOKEN "your-permit-token"
appwrite functions createVariable get_courses PERMIT_ENV "your-permit-environment"
appwrite functions createVariable get_courses PERMIT_PDP_ADDRESS "your-permit-pdp-address"
```

Repeat for all functions.

5. Deploy the function code:

```bash
appwrite functions deployments create get_courses --code ./get_courses.go
```

Repeat for all functions.

### Environment Variables for the Frontend

Create a `.env.local` file in the frontend directory with the following variables:

```bash
NEXT_PUBLIC_APPWRITE_ENDPOINT=https://your-appwrite-instance.com/v1
NEXT_PUBLIC_APPWRITE_PROJECT_ID=your-project-id
NEXT_PUBLIC_APPWRITE_DATABASE_ID=your-database-id
```

### Running the Frontend

```bash
cd frontend
pnpm install
pnpm run dev
```

## Authorization Flow

1. User logs in through Appwrite authentication.
2. The frontend gets the user's role and ID.
3. When the user tries to access a resource (e.g., a course), the frontend calls an Appwrite function.
4. The Go function checks with Permit.io to see if the user has the required permissions.
5. Permit.io evaluates the request based on the user's role, attributes, and the resource's attributes.
6. The function returns the appropriate response based on the authorization decision.

## Appwrite Collections

### Users Collection

- `id`: Unique identifier
- `name`: User's name
- `email`: User's email
- `role`: User's role (student, teacher, admin)

### Courses Collection

- `id`: Unique identifier
- `title`: Course title
- `description`: Course description
- `teacherId`: ID of the teacher who created the course
- `studentIds`: Array of student IDs enrolled in the course

### Assignments Collection

- `id`: Unique identifier
- `title`: Assignment title
- `description`: Assignment description
- `courseId`: ID of the course the assignment belongs to
- `dueDate`: Due date for the assignment

### Submissions Collection

- `id`: Unique identifier
- `assignmentId`: ID of the assignment
- `studentId`: ID of the student who submitted
- `content`: Submission content
- `submittedAt`: Submission date
- `grade`: Grade (0-100)
- `feedback`: Teacher feedback
