package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/dongri/phonenumber"
	"github.com/google/uuid"
)

type User struct {
	// ID is the unique identifier of the user.
	// It is automatically generated when the user is created.
	ID uuid.UUID `json:"id"`
	// Name is the first name of the user.
	Name string `json:"name,omitempty"`
	// LastName is the last name of the user.
	LastName string `json:"lastname,omitempty"`
	// PhoneNumber is the Ethiopian phone number of the user.
	PhoneNumber string `json:"phone_number,omitempty"`
	// Role is the user's role (admin, manager, user)
	Role string `json:"role,omitempty"`
	// Status is the user's account status (PENDING, ACTIVE, INACTIVE)
	Status string `json:"status,omitempty"`
	// JobTitle is the user's job title
	JobTitle *string `json:"job_title,omitempty"`
	// Education is the user's education background
	Education *string `json:"education,omitempty"`
	// MarriageStatus is the user's marriage status
	MarriageStatus *string `json:"marriage_status,omitempty"`
	// PartnerName is the name of the user's partner (spouse)
	PartnerName *string `json:"partner_name,omitempty"`
	// ChildrensName is a list of children's names
	ChildrensName []string `json:"childrens_name,omitempty"`
	// TelegramID is the optional Telegram ID of the user.
	TelegramID *string `json:"telegram_id,omitempty"`
	// CreatedAt is the time when the user is created.
	// It is automatically set when the user is created.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// DeletedAt is the time the user was deleted.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// UpdatedAt is the time the user was last updated.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type RegisterUser struct {
	// Name is the first name of the user.
	Name string `json:"name"`
	// LastName is the last name of the user.
	LastName string `json:"lastname"`
	// PhoneNumber is the Ethiopian phone number (+251...).
	PhoneNumber string `json:"phone_number"`
	// Password is the user's password.
	Password string `json:"password"`
	// Role is the user's role (admin, manager, user) - defaults to 'user'
	Role string `json:"role,omitempty"`
	// TelegramID is the optional Telegram ID of the user.
	TelegramID *string `json:"telegram_id,omitempty"`
}

func (u RegisterUser) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Name, validation.Required.Error("name is required"), validation.Length(2, 50).Error("name must be between 2 and 50 characters")),
		validation.Field(&u.LastName, validation.Required.Error("lastname is required"), validation.Length(2, 50).Error("lastname must be between 2 and 50 characters")),
		validation.Field(&u.PhoneNumber, 
			validation.Required.Error("phone number is required"),
			validation.By(u.validateEthiopianPhoneNumber)),
		validation.Field(&u.Password, 
			validation.Required.Error("password is required"), 
			validation.Length(6, 100).Error("password must be at least 6 characters")),
		validation.Field(&u.Role, validation.In("admin", "manager", "user", "").Error("role must be one of: admin, manager, user")),
	)
}

// validateEthiopianPhoneNumber validates and formats Ethiopian phone numbers
func (u RegisterUser) validateEthiopianPhoneNumber(value interface{}) error {
	phoneStr, ok := value.(string)
	if !ok {
		return validation.NewError("validation_phone_invalid_type", "phone number must be a string")
	}
	
	// Parse and validate the phone number for Ethiopia (ET)
	parsedNumber := phonenumber.Parse(phoneStr, "ET")
	
	// Check if the parsed number is empty (invalid)
	if parsedNumber == "" {
		return validation.NewError("validation_phone_invalid", "phone number must be a valid Ethiopian number (e.g., +251912345678, 0912345678, 912345678)")
	}
	
	// Verify it's actually an Ethiopian number (should start with 251)
	if len(parsedNumber) < 12 || parsedNumber[:3] != "251" {
		return validation.NewError("validation_phone_not_ethiopian", "phone number must be a valid Ethiopian number")
	}
	
	return nil
}

// NormalizePhoneNumber returns the phone number in E.164 format for storage
func (u RegisterUser) NormalizePhoneNumber() string {
	return phonenumber.Parse(u.PhoneNumber, "ET")
}

// GetRole returns the user role, defaulting to "user" if empty
func (u RegisterUser) GetRole() string {
	if u.Role == "" {
		return "user"
	}
	return u.Role
}

type UpdateUser struct {
	// Name is the first name of the user.
	Name *string `json:"name,omitempty"`
	// LastName is the last name of the user.
	LastName *string `json:"lastname,omitempty"`
	// PhoneNumber is the Ethiopian phone number of the user.
	PhoneNumber *string `json:"phone_number,omitempty"`
	// JobTitle is the user's job title
	JobTitle *string `json:"job_title,omitempty"`
	// Education is the user's education background
	Education *string `json:"education,omitempty"`
	// MarriageStatus is the user's marriage status (single, married, divorced, widowed)
	MarriageStatus *string `json:"marriage_status,omitempty"`
	// PartnerName is the name of the user's partner (spouse)
	PartnerName *string `json:"partner_name,omitempty"`
	// ChildrensName is a list of children's names
	ChildrensName *[]string `json:"childrens_name,omitempty"`
	// TelegramID is the optional Telegram ID of the user.
	TelegramID *string `json:"telegram_id,omitempty"`
}

func (u UpdateUser) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Name, validation.When(u.Name != nil, validation.Length(2, 50).Error("name must be between 2 and 50 characters"))),
		validation.Field(&u.LastName, validation.When(u.LastName != nil, validation.Length(2, 50).Error("lastname must be between 2 and 50 characters"))),
		// Keep field reference but validator handles pointer dereferencing internally
		validation.Field(&u.PhoneNumber, validation.When(u.PhoneNumber != nil, validation.By(u.validateEthiopianPhoneNumber))),
		validation.Field(&u.JobTitle, validation.When(u.JobTitle != nil && *u.JobTitle != "", validation.Length(1, 100).Error("job title must be between 1 and 100 characters"))),
		validation.Field(&u.Education, validation.When(u.Education != nil && *u.Education != "", validation.Length(1, 200).Error("education must be between 1 and 200 characters"))),
		validation.Field(&u.MarriageStatus, validation.When(u.MarriageStatus != nil && *u.MarriageStatus != "", validation.In("single", "married", "divorced", "widowed").Error("marriage status must be one of: single, married, divorced, widowed"))),
		validation.Field(&u.PartnerName, validation.When(u.PartnerName != nil && *u.PartnerName != "", validation.Length(1, 100).Error("partner name must be between 1 and 100 characters"))),
		// Validate children names with a custom validator that dereferences pointers and enforces limits
		validation.Field(&u.ChildrensName, validation.When(u.ChildrensName != nil, validation.By(u.validateChildrenNames))),
	)
}

// validateEthiopianPhoneNumber validates and formats Ethiopian phone numbers for UpdateUser
func (u UpdateUser) validateEthiopianPhoneNumber(value interface{}) error {
	var phoneStr string
	
	// Handle the fact that ozzo-validation passes **string when we use &u.PhoneNumber
	switch v := value.(type) {
	case string:
		phoneStr = v
	case *string:
		if v == nil {
			return validation.NewError("validation_phone_invalid_type", "phone number must be a string")
		}
		phoneStr = *v
	case **string:
		// This is what we get when validation.Field(&u.PhoneNumber, ...) is used
		if v == nil || *v == nil {
			return validation.NewError("validation_phone_invalid_type", "phone number must be a string")
		}
		phoneStr = **v
	default:
		return validation.NewError("validation_phone_invalid_type", "phone number must be a string")
	}
	
	// Parse and validate the phone number for Ethiopia (ET)
	parsedNumber := phonenumber.Parse(phoneStr, "ET")
	
	// Check if the parsed number is empty (invalid)
	if parsedNumber == "" {
		return validation.NewError("validation_phone_invalid", "phone number must be a valid Ethiopian number (e.g., +251912345678, 0912345678, 912345678)")
	}
	
	// Verify it's actually an Ethiopian number (should start with 251)
	if len(parsedNumber) < 12 || parsedNumber[:3] != "251" {
		return validation.NewError("validation_phone_not_ethiopian", "phone number must be a valid Ethiopian number")
	}
	
	return nil
}

// validateChildrenNames validates childrens_name allowing pointers and enforcing constraints
func (u UpdateUser) validateChildrenNames(value interface{}) error {
	var names []string

	// ozzo-validation may pass **[]string when using Field(&u.ChildrensName)
	switch v := value.(type) {
	case []string:
		names = v
	case *[]string:
		if v == nil {
			return nil // nothing to validate
		}
		names = *v
	case **[]string:
		if v == nil || *v == nil {
			return nil
		}
		names = **v
	default:
		return validation.NewError("validation_children_invalid_type", "childrens_name must be an array of strings")
	}

	// Max 10 children
	if len(names) > 10 {
		return validation.NewError("validation_children_too_many", "maximum 10 children names allowed")
	}

	// Validate each child name length between 1 and 50
	for _, n := range names {
		if l := len(n); l < 1 || l > 50 {
			return validation.NewError("validation_children_name_length", "each child name must be between 1 and 50 characters")
		}
	}

	return nil
}

// NormalizePhoneNumber returns the phone number in E.164 format for storage (for UpdateUser)
func (u UpdateUser) NormalizePhoneNumber() *string {
	if u.PhoneNumber == nil {
		return nil
	}
	normalized := phonenumber.Parse(*u.PhoneNumber, "ET")
	return &normalized
}

// ChangePasswordRequest represents the request payload for changing user password
type ChangePasswordRequest struct {
	// CurrentPassword is the user's current password for verification
	CurrentPassword string `json:"current_password"`
	// NewPassword is the new password the user wants to set
	NewPassword string `json:"new_password"`
}

func (c ChangePasswordRequest) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.CurrentPassword, 
			validation.Required.Error("current password is required")),
		validation.Field(&c.NewPassword, 
			validation.Required.Error("new password is required"), 
			validation.Length(6, 100).Error("new password must be at least 6 characters")),
	)
}

// UserListResponse represents a paginated list of users
type UserListResponse struct {
	Users    []User `json:"users"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// UserStatusUpdateRequest represents the request payload for updating user status
type UserStatusUpdateRequest struct {
	// Status is the new status to set (ACTIVE, INACTIVE)
	Status string `json:"status"`
}

func (u UserStatusUpdateRequest) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Status, 
			validation.Required.Error("status is required"),
			validation.In("ACTIVE", "INACTIVE").Error("status must be ACTIVE or INACTIVE")),
	)
}
