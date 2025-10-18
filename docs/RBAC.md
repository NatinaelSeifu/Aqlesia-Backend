# Role-Based Access Control (RBAC) with Open Policy Agent

This project implements fine-grained authorization using **Open Policy Agent (OPA)** for role-based access control.

## User Roles

The system supports three user roles with different permission levels:

### Admin
- **Full access** to all system resources
- Can perform all operations on users (create, read, update, delete, list)
- **Permissions**: `users:read`, `users:write`, `users:delete`, `users:list`

### Manager  
- **Limited administrative access**
- Can list, read, and update users but cannot delete them
- **Permissions**: `users:read`, `users:write`, `users:list`

### User (Default)
- **Basic access** for regular users
- Can read and update their own profile
- **Permissions**: `users:read_own`, `users:write_own`

## API Endpoint Permissions

| Endpoint | Method | Required Permission | Role Access |
|----------|--------|-------------------|-------------|
| `POST /auth/register` | POST | None (registration) | All users |
| `GET /users` | GET | `users:list` | Admin, Manager |
| `GET /users/{id}` | GET | `users:read` or `users:read_own` | Admin, Manager, or Own Profile |
| `PATCH /users/{id}` | PATCH | `users:write` or `users:write_own` | Admin, Manager, or Own Profile |
| `DELETE /users/{id}` | DELETE | `users:delete` | Admin only |

## OPA Policy Rules

The authorization policies are defined in `internal/opa/policies/rbac.rego`:

1. **Default Deny**: All actions are denied by default
2. **Role-Based Permissions**: Users inherit permissions based on their role
3. **Self-Access**: Users can always read their own profile
4. **Admin Override**: Admin users can perform any action

## Usage Examples

### Authentication Flow
1. **Register**: `POST /v1/auth/register` (no auth required)
2. **Login**: `POST /v1/auth/login` with phone number and password
3. **Access Protected Routes**: Include `Authorization: Bearer <token>` header

### Role Assignment
Users are assigned roles during registration:
- Default role: `user`
- Admins must be created with `role: "admin"`
- Managers must be created with `role: "manager"`

### Example Requests

#### User registration:
```bash
curl -X POST http://localhost:8000/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"John","lastname":"Doe","phone_number":"+251912345678","password":"secret123"}'
```

#### Admin listing all users:
```bash
curl -H "Authorization: Bearer <admin_token>" \
     GET http://localhost:8000/v1/users
```

#### User updating their own profile:
```bash
curl -H "Authorization: Bearer <user_token>" \
     -H "Content-Type: application/json" \
     -d '{"job_title": "Software Engineer", "education": "Computer Science", "marriage_status": "single", "childrens_name": ["Alice", "Bob"]}' \
     PATCH http://localhost:8000/v1/users/<their_user_id>
```

#### Manager updating any user:
```bash
curl -H "Authorization: Bearer <manager_token>" \
     -H "Content-Type: application/json" \
     -d '{"name": "Updated Name", "phone_number": "+251987654321"}' \
     PATCH http://localhost:8000/v1/users/<user_id>
```

## Error Responses

- **401 Unauthorized**: Invalid or missing JWT token
- **403 Forbidden**: Valid token but insufficient permissions
- **400 Bad Request**: Invalid request format or parameters

## Policy Development

The OPA policies can be extended to support:
- Resource-based permissions
- Time-based access control  
- Attribute-based access control (ABAC)
- Dynamic permission assignment

To modify policies, edit `internal/opa/policies/rbac.rego` and restart the application.

## Testing Authorization

Use the Swagger UI at `http://localhost:8000/v1/swagger/index.html` to test different roles and permissions interactively.
