# GoBook: Resource Booking & Sale Platform

A comprehensive booking and sale platform built with Go, allowing users to list resources (rooms, equipment, services) for booking or direct sale. The system supports multi-image uploads, location-based filtering, reviews, and comprehensive admin management.

## Team

**Group**: SE-2425  
**Members**: 
- **Lenara Symbatkyzy** - Resources module (CRUD), Notifications, Admin panel, Frontend templates
- **Nursalim Onalbayev** - Bookings module, Payments, Authentication, Database design

## How It Works

### Architecture

GoBook is built as a **modular monolith** using Go's standard `net/http` package. The system follows a clean architecture pattern:

1. **HTTP Layer** (`cmd/api/main.go`): Entry point, route definitions, middleware chain
2. **Module Layer** (`internal/modules/`): Business logic handlers for auth, resources, bookings
3. **Database Layer** (`internal/database/`): SQLite connection with safe concurrent access
4. **Service Layer** (`internal/services/`): Background workers for async tasks
5. **Frontend** (`web/templates/`, `web/static/`): HTML templates with Go templating

### Core Features

#### 1. User Management (Nursalim)
- Registration and login with session-based authentication
- Password hashing using bcrypt
- User profile management with payment details
- Role-based access (user/admin)

#### 2. Resource Management (Lenara)
- Create resources with multiple images
- Two types: **booking** (hourly rate) or **sale** (fixed price)
- Location-based resources (Kazakhstan cities)
- Category classification
- Resource creator can view all bookings for their resources

#### 3. Booking System (Nursalim)
- Create bookings with conflict detection
- Time-based bookings for hourly resources
- Direct purchase for sale-type resources
- Price calculation (hourly × duration or fixed price)
- Booking confirmation by resource creator
- Image attachments to bookings

#### 4. Reviews System (Lenara)
- Users can leave reviews on resources
- Rating system (1-5 stars)
- Review display on resource detail page
- Update existing reviews

#### 5. Dashboard & Filtering (Lenara)
- Filter by location, category, booking type, price range
- Resource cards with images
- Quick access to view details, edit, or book

#### 6. Admin Panel (Lenara)
- User management (list, edit, delete, reset password)
- Audit log for all system actions
- Resource and booking management
- Cascade deletion (deleting user deletes their bookings/resources)

#### 7. Background Workers (Nursalim)
- Notification processor (every 30 seconds)
- Booking cleanup (marks expired bookings)
- Payment processing simulation (every 2 minutes)

### Database Schema

SQLite database with the following tables:

- **users**: User accounts, authentication, payment details
- **resources**: Bookable/sellable resources with images, location, pricing
- **bookings**: Booking records with conflict detection
- **payments**: Payment transactions
- **notifications**: System notifications queue
- **audit_logs**: Audit trail for admin actions
- **reviews**: User reviews for resources

### Technology Stack

- **Backend**: Go 1.25+ with `net/http`
- **Database**: SQLite (using `modernc.org/sqlite` - pure Go, no CGO)
- **Templating**: Go `html/template` package
- **Authentication**: Session cookies + bcrypt
- **Concurrency**: Goroutines, channels, mutexes for safe database access

## Quick Start

### Prerequisites
- Go 1.25 or later
- No CGO required (pure Go SQLite driver)

### Installation

1. **Clone the repository**
   ```bash
   cd ADP_Project
   ```

2. **Install dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Run the application**
   ```bash
   go run cmd/api/main.go
   ```

4. **Access the application**
   - Web Interface: http://localhost:8081
   - Database file `gobook.db` is created automatically
   - **Default admin account** (created automatically):
     - Email: `admin@gobook.com`
     - Password: `admin123`

## Key Features

### Resource Types

1. **Booking Type**: Hourly-based rental
   - Price per hour
   - Requires start/end time
   - Total price = hourly rate × duration

2. **Sale Type**: Fixed-price purchase
   - One-time payment
   - No time required
   - Immediate purchase

### Location System

- Predefined list of Kazakhstan cities (in English)
- Default location: "Astana"
- Filter resources by location on dashboard

### Image Management

- Multiple images per resource
- Image gallery with thumbnail navigation
- Upload during resource creation/editing
- Delete individual images

### Review System

- Leave reviews on resource detail page
- Rating from 1-5 stars
- Display username and comment
- Update existing reviews

### Filtering

Dashboard supports filtering by:
- **Location**: All Kazakhstan cities
- **Category**: Resource category
- **Type**: Booking or Sale
- **Price Range**: Min/max price

## API Endpoints

### Authentication
- `GET /login` - Login page
- `POST /login` - Login (form)
- `GET /register` - Registration page
- `POST /register` - Register (form)
- `POST /api/auth/register` - Register (JSON API)
- `POST /api/auth/login` - Login (JSON API)

### Resources
- `GET /dashboard` - Resource dashboard with filters
- `GET /resources/create` - Create resource page
- `POST /resources/create` - Create resource (form)
- `GET /resources/view?id=X` - View resource details
- `GET /resources/edit?id=X` - Edit resource page
- `POST /resources/update` - Update resource (form)
- `GET /resources/bookings?resource_id=X` - View bookings for resource
- `POST /resources/review` - Create/update review
- `DELETE /api/resources/delete?id=X` - Delete resource (JSON)

### Bookings
- `GET /bookings/create` - Create booking page
- `POST /bookings/create` - Create booking (form)
- `GET /bookings/my` - My bookings list
- `GET /api/bookings` - Get user bookings (JSON)
- `POST /api/bookings` - Create booking (JSON)

### Admin
- `GET /admin/users` - User management
- `GET /admin/users/edit?id=X` - Edit user
- `POST /admin/users/update` - Update user
- `POST /admin/users/delete?id=X` - Delete user
- `GET /admin/audit` - Audit log
- `POST /admin/resources/delete?id=X` - Delete resource
- `POST /admin/bookings/delete?id=X` - Delete booking

### Profile
- `GET /profile` - User profile page
- `POST /profile/update` - Update profile

## Project Structure

```
ADP_Project/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point, routes, middleware
├── internal/
│   ├── database/
│   │   ├── db.go               # Database initialization, safe access
│   │   ├── migrations.go       # Schema migrations
│   │   └── admin.go             # Default admin creation
│   ├── models/
│   │   └── types.go            # Data models (User, Resource, Booking, etc.)
│   ├── middleware/
│   │   └── middleware.go       # HTTP middleware (auth, logging, CORS)
│   ├── modules/
│   │   ├── auth/
│   │   │   ├── handler.go      # Registration, login
│   │   │   ├── admin.go        # Admin user management
│   │   │   ├── audit.go        # Audit log
│   │   │   ├── edit.go         # User editing
│   │   │   ├── profile.go     # User profile
│   │   │   └── form_helper.go # Form parsing helper
│   │   ├── resource/
│   │   │   ├── handler.go      # Resource CRUD, dashboard
│   │   │   ├── edit.go         # Resource editing
│   │   │   ├── view.go         # Resource detail view
│   │   │   ├── bookings.go     # View resource bookings
│   │   │   ├── images.go       # Image upload handling
│   │   │   ├── delete_image.go # Image deletion
│   │   │   ├── reviews.go      # Review system
│   │   │   └── locations.go    # Kazakhstan cities list
│   │   └── booking/
│   │       ├── handler.go      # Booking creation, conflict detection
│   │       ├── list.go         # User's bookings
│   │       ├── edit.go         # Booking editing, confirmation
│   │       └── form_helper.go # Form parsing helper
│   └── services/
│       └── worker.go           # Background workers (notifications, cleanup, payments)
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css       # Global styles
│   │   └── js/
│   │       └── modal.js       # Pop-up modal system
│   └── templates/
│       ├── layout.html         # Base template
│       ├── login.html          # Login page
│       ├── register.html       # Registration page
│       ├── dashboard.html      # Resource dashboard with filters
│       ├── create_resource.html # Resource creation form
│       ├── edit_resource.html  # Resource editing form
│       ├── view_resource.html  # Resource detail with reviews
│       ├── create_booking.html # Booking creation form
│       ├── edit_booking.html   # Booking editing form
│       ├── my_bookings.html    # User's bookings list
│       ├── resource_bookings.html # Resource creator's bookings view
│       ├── profile.html        # User profile page
│       ├── admin_users.html    # Admin user management
│       ├── admin_audit.html    # Audit log page
│       └── edit_user.html      # User editing form
├── go.mod                      # Go module dependencies
└── README.md                   # This file
```

## Concurrency & Performance

### Safe Database Access
- SQLite WAL mode enabled for concurrent reads
- `busy_timeout=1000` for automatic retry on locks
- Connection pool: max 10 open, 5 idle connections
- Mutex protection for write operations

### Background Processing
- **Notification Worker**: Processes pending notifications every 30 seconds
- **Cleanup Worker**: Marks expired bookings as completed every 5 minutes
- **Payment Worker**: Simulates payment processing every 2 minutes
- All workers use goroutines and channels for async processing

## Development Notes

### Code Style
- All user-facing messages in English
- Russian comments (lowercase, lazy style) for code explanation
- Clean separation of concerns (handlers, database, services)

### Database Migrations
- Automatic schema migrations on startup
- Adds new columns if they don't exist (images, location, booking_type, etc.)
- No manual migration scripts needed

### Error Handling
- Pop-up modals for user feedback (success/error messages)
- URL parameter-based error passing
- Comprehensive error logging

## Testing

1. **Start the server**: `go run cmd/api/main.go`
2. **Access**: http://localhost:8081
3. **Login as admin**: `admin@gobook.com` / `admin123`
4. **Create resources**: Add resources with images, set location and type
5. **Create bookings**: Book resources or purchase sale items
6. **Leave reviews**: Review resources on detail page
7. **Test filters**: Filter dashboard by location, category, price

## Troubleshooting

### "User already exists" error
- Use admin panel to delete existing user
- Or delete `gobook.db` and restart (database will be recreated)

### Database locked errors
- System uses WAL mode and busy_timeout for automatic retry
- If persistent, restart the server

### Images not uploading
- Check file size (max 32MB per upload)
- Ensure `web/uploads/` directory exists and is writable

---

**Project Status**: Complete and ready for demonstration.
