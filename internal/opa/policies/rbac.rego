package rbac

import rego.v1

# Default deny
default allow := false

# Role definitions
roles := {
    "admin": {
        "permissions": [
            "users:read",
            "users:write", 
            "users:delete",
            "users:list",
            "appointments:read",
            "appointments:write",
            "appointments:delete",
            "appointments:list",
            "appointments:complete",
            "appointments:cancel",
            "appointments:stats",
            "appointments:list_available",
            "available_dates:read",
            "available_dates:list",
            "available_dates:create",
            "available_dates:update",
            "available_dates:delete",
            "communions:list",
            "communions:manage",
            "communions:delete",
            "questions:list",
            "questions:read",
            "questions:respond",
            "questions:update",
            "questions:delete",
            "questions:stats"
        ]
    },
    "manager": {
        "permissions": [
            "users:read",
            "users:write",
            "users:list",
            "appointments:read",
            "appointments:write",
            "appointments:list",
            "appointments:complete",
            "appointments:cancel",
            "appointments:stats",
            "appointments:list_available",
            "available_dates:read",
            "available_dates:list",
            "available_dates:create",
            "available_dates:update",
            "available_dates:delete",
            "questions:list",
            "questions:read",
            "questions:respond",
            "questions:update",
            "questions:stats"
        ]
    },
    "user": {
        "permissions": [
            "users:read_own",
            "users:write_own",
            "appointments:create",
            "appointments:read_own",
            "appointments:write_own",
            "appointments:cancel_own",
            "appointments:list_available",
            "available_dates:read",
            "available_dates:list",
            "communions:create",
            "communions:read_own",
            "communions:write_own",
            "questions:create",
            "questions:read_own",
            "questions:update_own"
        ]
    }
}

# Allow if user has required permission for the action
allow if {
    # Get user role from input
    user_role := input.user.role
    
    # Get required permission from input
    required_permission := input.permission
    
    # Check if user's role has the required permission
    required_permission in roles[user_role].permissions
}

# Allow users to read their own profile regardless of role
allow if {
    input.permission == "users:read_own"
    input.user.id == input.resource.user_id
}

# Allow users to update their own profile
allow if {
    input.permission == "users:write_own"
    input.user.id == input.resource.user_id
}

# Allow users to read their own appointments
allow if {
    input.permission == "appointments:read_own"
    input.user.id == input.resource.user_id
}

# Allow users to update their own appointments
allow if {
    input.permission == "appointments:write_own"
    input.user.id == input.resource.user_id
}

# Allow users to cancel their own appointments
allow if {
    input.permission == "appointments:cancel_own"
    input.user.id == input.resource.user_id
}

# Admin can do everything
allow if {
    input.user.role == "admin"
}

# Helper function to check if user can list users
can_list_users if {
    allow with input as {
        "user": input.user,
        "permission": "users:list"
    }
}

# Helper function to check if user can create users  
can_create_users if {
    allow with input as {
        "user": input.user,
        "permission": "users:write"
    }
}

# Helper function to check if user can update users
can_update_users if {
    allow with input as {
        "user": input.user,
        "permission": "users:write"
    }
}

# Helper function to check if user can delete users
can_delete_users if {
    allow with input as {
        "user": input.user,
        "permission": "users:delete"
    }
}

# Helper function to check if user can read specific user
can_read_user if {
    allow with input as {
        "user": input.user,
        "permission": "users:read",
        "resource": input.resource
    }
}

# Allow users to read their own communion requests
allow if {
    input.permission == "communions:read_own"
    input.user.id == input.resource.user_id
}

# Allow users to update their own pending communion requests
allow if {
    input.permission == "communions:write_own"
    input.user.id == input.resource.user_id
}

# Helper function to check if user can list all communions
can_list_communions if {
    allow with input as {
        "user": input.user,
        "permission": "communions:list"
    }
}

# Helper function to check if user can manage communions (approve/reject)
can_manage_communions if {
    allow with input as {
        "user": input.user,
        "permission": "communions:manage"
    }
}

# Helper function to check if user can delete communions
can_delete_communions if {
    allow with input as {
        "user": input.user,
        "permission": "communions:delete"
    }
}
