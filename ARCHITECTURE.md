Architecture & Design

Modular Monolith
The system utilizes a Modular Monolith architecture to ensure the logic is strictly separated into layers, allowing for future migration to microservices.

Diagrams
Use-Case Diagram 
Actors: Client and Administrator.
Key Use Cases: Browse Services, Create Booking, Cancel Booking, Manage Inventory, View Reports.

UML Class Diagram 
The architecture includes core components such as:
Server: Router mux and Config management.
BookingHandler: Handling Create and List requests.
BookingService: Availability checks and total calculations.
Repository: Saving data and finding overlaps.

Entity Relationship Diagram (ERD) 
The database schema consists of:
USER: id, email, password_hash, role.
RESOURCE: id, name, category, hourly_rate.
BOOKING: id, user_id, resource_id, start_time, end_time, status.