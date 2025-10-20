package user

import (
	"aqlesia/internal/constants/dbinstance"
	"aqlesia/internal/constants/errors"
	"aqlesia/internal/constants/model/db"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/storage"
	"aqlesia/platform/logger"
	"context"
	"database/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type user struct {
	db  dbinstance.DBInstance
	log logger.Logger
}

func Init(db dbinstance.DBInstance, log logger.Logger) storage.User {
	return &user{
		db:  db,
		log: log,
	}
}
func (u *user) Create(ctx context.Context, param dto.RegisterUser) (*dto.User, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(param.Password), bcrypt.DefaultCost)
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not hash password")
		u.log.Error(ctx, "unable to hash password", zap.Error(err))
		return nil, err
	}

	// Convert telegram_id to sql.NullString
	var telegramID sql.NullString
	if param.TelegramID != nil {
		telegramID = sql.NullString{String: *param.TelegramID, Valid: true}
	}

	// Normalize phone number to E.164 format for storage
	normalizedPhone := param.NormalizePhoneNumber()
	
	user, err := u.db.CreateUser(ctx, db.CreateUserParams{
		Name:        param.Name,
		Lastname:    param.LastName,
		PhoneNumber: normalizedPhone,
		Password:    string(hashedPassword),
		Role:        param.GetRole(),
		TelegramID:  telegramID,
		Status:      "PENDING", // Set new users to pending status
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not create user")
		u.log.Error(ctx, "unable to create user", zap.Error(err), zap.Any("user", param))
		return nil, err
	}

	// Convert optional fields back for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus *string
	if user.TelegramID.Valid {
		responseTelegramID = &user.TelegramID.String
	}
	if user.JobTitle.Valid {
		responseJobTitle = &user.JobTitle.String
	}
	if user.Education.Valid {
		responseEducation = &user.Education.String
	}
	if user.MarriageStatus.Valid {
		responseMarriageStatus = &user.MarriageStatus.String
	}

	return &dto.User{
		ID:             user.ID,
		Name:           user.Name,
		LastName:       user.Lastname,
		PhoneNumber:    user.PhoneNumber,
		Role:           user.Role,
		Status:         string(user.Status), // Use actual status from database
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		ChildrensName:  user.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}, nil
}

func (u *user) Update(ctx context.Context, id uuid.UUID, param dto.UpdateUser) (*dto.User, error) {
	// First, get the current user data to preserve unspecified fields
	currentUser, err := u.db.GetUser(ctx, id)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not read current user")
		u.log.Error(ctx, "unable to get current user for update", zap.Error(err), zap.String("user-id", id.String()))
		return nil, err
	}

	// Handle phone number normalization
	var normalizedPhone string
	if param.PhoneNumber != nil {
		normalizedPhonePtr := param.NormalizePhoneNumber()
		if normalizedPhonePtr != nil {
			normalizedPhone = *normalizedPhonePtr
		}
	}

	// Handle optional fields: 
	// - If field is provided (not nil), use the provided value (empty string becomes NULL)
	// - If field is not provided (nil), set to NULL
	var jobTitle, education, marriageStatus, partnerName, telegramID sql.NullString
	
	if param.JobTitle != nil {
		if *param.JobTitle == "" {
			jobTitle = sql.NullString{String: "", Valid: false} // Set to NULL
		} else {
			jobTitle = sql.NullString{String: *param.JobTitle, Valid: true}
		}
	} else {
		// Field not provided - set to NULL
		jobTitle = sql.NullString{String: "", Valid: false}
	}
	
	if param.Education != nil {
		if *param.Education == "" {
			education = sql.NullString{String: "", Valid: false} // Set to NULL
		} else {
			education = sql.NullString{String: *param.Education, Valid: true}
		}
	} else {
		// Field not provided - set to NULL
		education = sql.NullString{String: "", Valid: false}
	}
	
	if param.MarriageStatus != nil {
		if *param.MarriageStatus == "" {
			marriageStatus = sql.NullString{String: "", Valid: false} // Set to NULL
		} else {
			marriageStatus = sql.NullString{String: *param.MarriageStatus, Valid: true}
		}
	} else {
		// Field not provided - set to NULL
		marriageStatus = sql.NullString{String: "", Valid: false}
	}
	
	if param.PartnerName != nil {
		if *param.PartnerName == "" {
			partnerName = sql.NullString{String: "", Valid: false} // Set to NULL
		} else {
			partnerName = sql.NullString{String: *param.PartnerName, Valid: true}
		}
	} else {
		// Field not provided - set to NULL
		partnerName = sql.NullString{String: "", Valid: false}
	}
	
	if param.TelegramID != nil {
		if *param.TelegramID == "" {
			telegramID = sql.NullString{String: "", Valid: false} // Set to NULL
		} else {
			telegramID = sql.NullString{String: *param.TelegramID, Valid: true}
		}
	} else {
		// Field not provided - set to NULL
		telegramID = sql.NullString{String: "", Valid: false}
	}

	// Handle required fields, preserving current values if not provided
	// Required fields (name, lastname, phone) cannot be set to empty - preserve current value
	name := currentUser.Name
	lastName := currentUser.Lastname
	phoneNumber := currentUser.PhoneNumber
	if param.Name != nil && *param.Name != "" {
		name = *param.Name
	}
	if param.LastName != nil && *param.LastName != "" {
		lastName = *param.LastName
	}
	if param.PhoneNumber != nil && *param.PhoneNumber != "" {
		phoneNumber = normalizedPhone
	}

	// Handle children names array - set to empty if not provided
	var childrensName []string
	if param.ChildrensName != nil {
		childrensName = *param.ChildrensName
	} else {
		// Field not provided - set to empty array (equivalent to NULL for arrays)
		childrensName = []string{}
	}

	user, err := u.db.UpdateUser(ctx, db.UpdateUserParams{
		Name:           name,
		Lastname:       lastName,
		PhoneNumber:    phoneNumber,
		JobTitle:       jobTitle,
		Education:      education,
		MarriageStatus: marriageStatus,
		PartnerName:    partnerName,
		ChildrensName:  childrensName,
		TelegramID:     telegramID,
		ID:             id,
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not update user")
		u.log.Error(ctx, "unable to update user", zap.Error(err), zap.Any("user", param), zap.String("user-id", id.String()))
		return nil, err
	}

	// Convert optional fields back for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
	if user.TelegramID.Valid {
		responseTelegramID = &user.TelegramID.String
	}
	if user.JobTitle.Valid {
		responseJobTitle = &user.JobTitle.String
	}
	if user.Education.Valid {
		responseEducation = &user.Education.String
	}
	if user.MarriageStatus.Valid {
		responseMarriageStatus = &user.MarriageStatus.String
	}
	if user.PartnerName.Valid {
		responsePartnerName = &user.PartnerName.String
	}

	return &dto.User{
		ID:             user.ID,
		Name:           user.Name,
		LastName:       user.Lastname,
		PhoneNumber:    user.PhoneNumber,
		Role:           user.Role,
		Status:         string(user.Status), // Use actual status from database
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		PartnerName:    responsePartnerName,
		ChildrensName:  user.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}, nil
}

func (u *user) Get(ctx context.Context, id uuid.UUID) (*dto.User, error) {
	user, err := u.db.GetUser(ctx, id)
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not read user")
		u.log.Error(ctx, "unable to get user", zap.Error(err), zap.String("user-id", id.String()))
		return nil, err
	}

	// Convert optional fields back for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
	if user.TelegramID.Valid {
		responseTelegramID = &user.TelegramID.String
	}
	if user.JobTitle.Valid {
		responseJobTitle = &user.JobTitle.String
	}
	if user.Education.Valid {
		responseEducation = &user.Education.String
	}
	if user.MarriageStatus.Valid {
		responseMarriageStatus = &user.MarriageStatus.String
	}
	if user.PartnerName.Valid {
		responsePartnerName = &user.PartnerName.String
	}

	return &dto.User{
		ID:             user.ID,
		Name:           user.Name,
		LastName:       user.Lastname,
		PhoneNumber:    user.PhoneNumber,
		Role:           user.Role,
		Status:         string(user.Status), // Use actual status from database
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		PartnerName:    responsePartnerName,
		ChildrensName:  user.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}, nil
}

func (u *user) GetAll(ctx context.Context, page, pageSize int) ([]dto.User, int64, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated users
	users, err := u.db.GetUsers(ctx, db.GetUsersParams{
		OffsetCount: int32(offset),
		LimitCount:  int32(pageSize),
	})
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not read users")
		u.log.Error(ctx, "unable to get users", zap.Error(err))
		return nil, 0, err
	}

	// Get total count
	total, err := u.db.CountUsers(ctx)
	if err != nil {
		err = errors.ErrReadError.Wrap(err, "could not count users")
		u.log.Error(ctx, "unable to count users", zap.Error(err))
		return nil, 0, err
	}

	// Convert database users to DTOs
	dtoUsers := make([]dto.User, len(users))
	for i, user := range users {
		var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
		if user.TelegramID.Valid {
			responseTelegramID = &user.TelegramID.String
		}
		if user.JobTitle.Valid {
			responseJobTitle = &user.JobTitle.String
		}
		if user.Education.Valid {
			responseEducation = &user.Education.String
		}
		if user.MarriageStatus.Valid {
			responseMarriageStatus = &user.MarriageStatus.String
		}
		if user.PartnerName.Valid {
			responsePartnerName = &user.PartnerName.String
		}

		dtoUsers[i] = dto.User{
			ID:             user.ID,
			Name:           user.Name,
			LastName:       user.Lastname,
			PhoneNumber:    user.PhoneNumber,
			Role:           user.Role,
			Status:         string(user.Status), // Use actual status from database
			JobTitle:       responseJobTitle,
			Education:      responseEducation,
			MarriageStatus: responseMarriageStatus,
			PartnerName:    responsePartnerName,
			ChildrensName:  user.ChildrensName,
			TelegramID:     responseTelegramID,
			CreatedAt:      user.CreatedAt,
			UpdatedAt:      user.UpdatedAt,
		}
	}
	return dtoUsers, total, nil
}

func (u *user) CheckUserExists(ctx context.Context, param dto.RegisterUser) (bool, error) {
	// Use normalized phone number for consistency
	normalizedPhone := param.NormalizePhoneNumber()
	count, err := u.db.UserByPhoneExists(ctx, normalizedPhone)
	if err != nil {
		err := errors.ErrReadError.Wrap(err, "could not read user")
		u.log.Error(ctx, "unable to read the user", zap.Error(err), zap.Any("user-phone", normalizedPhone))
		return false, err
	}

	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (u *user) GetUserByPhone(ctx context.Context, phoneNumber string) (*dto.User, error) {
	user, err := u.db.GetUserByPhone(ctx, phoneNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			err := errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User with this phone number not found", zap.String("phone", phoneNumber))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read user")
		u.log.Error(ctx, "unable to get user by phone", zap.Error(err), zap.String("phone", phoneNumber))
		return nil, err
	}

	// Convert optional fields for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
	if user.TelegramID.Valid {
		responseTelegramID = &user.TelegramID.String
	}
	if user.JobTitle.Valid {
		responseJobTitle = &user.JobTitle.String
	}
	if user.Education.Valid {
		responseEducation = &user.Education.String
	}
	if user.MarriageStatus.Valid {
		responseMarriageStatus = &user.MarriageStatus.String
	}
	if user.PartnerName.Valid {
		responsePartnerName = &user.PartnerName.String
	}

	return &dto.User{
		ID:             user.ID,
		Name:           user.Name,
		LastName:       user.Lastname,
		PhoneNumber:    user.PhoneNumber,
		Role:           user.Role,
		Status:         string(user.Status), // Use actual status from database
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		PartnerName:    responsePartnerName,
		ChildrensName:  user.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}, nil
}

func (u *user) GetUserByPhoneWithPassword(ctx context.Context, phoneNumber string) (*db.User, error) {
	user, err := u.db.GetUserByPhone(ctx, phoneNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			err := errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User with this phone number not found", zap.String("phone", phoneNumber))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read user")
		u.log.Error(ctx, "unable to get user by phone", zap.Error(err), zap.String("phone", phoneNumber))
		return nil, err
	}

	return &user, nil
}

func (u *user) DeleteUser(ctx context.Context, userId uuid.UUID) error {
	_, err := u.db.DeleteUser(ctx, userId)
	if err != nil {
		if err == sql.ErrNoRows {
			err := errors.ErrNoRecordFound.Wrap(err, "no record of user found")
			u.log.Info(ctx, "User with this id not found", zap.Error(err), zap.String("user-id", userId.String()))
			return err
		}
		err = errors.ErrWriteError.Wrap(err, "error deleting the user")
		u.log.Error(ctx, "unable to delete user data", zap.Error(err), zap.String("user-id", userId.String()))
		return err
	}
	return nil
}

func (u *user) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) (*dto.User, error) {
	// First, get the user with password to verify current password
	currentUser, err := u.db.GetUser(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			err := errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User not found for password change", zap.String("user-id", userID.String()))
			return nil, err
		}
		err = errors.ErrReadError.Wrap(err, "could not read user")
		u.log.Error(ctx, "unable to get user for password change", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(currentPassword))
	if err != nil {
		err = errors.ErrInvalidUserInput.New("current password is incorrect")
		u.log.Warn(ctx, "incorrect current password provided", zap.String("user-id", userID.String()))
		return nil, err
	}

	// Hash the new password
	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		err = errors.ErrWriteError.Wrap(err, "could not hash new password")
		u.log.Error(ctx, "unable to hash new password", zap.Error(err))
		return nil, err
	}

	// Update the password in the database
	updatedUser, err := u.db.ChangePassword(ctx, db.ChangePasswordParams{
		Password: string(hashedNewPassword),
		ID:       userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User not found during password update", zap.String("user-id", userID.String()))
			return nil, err
		}
		err = errors.ErrWriteError.Wrap(err, "could not update password")
		u.log.Error(ctx, "unable to update user password", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	// Convert optional fields back for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
	if updatedUser.TelegramID.Valid {
		responseTelegramID = &updatedUser.TelegramID.String
	}
	if updatedUser.JobTitle.Valid {
		responseJobTitle = &updatedUser.JobTitle.String
	}
	if updatedUser.Education.Valid {
		responseEducation = &updatedUser.Education.String
	}
	if updatedUser.MarriageStatus.Valid {
		responseMarriageStatus = &updatedUser.MarriageStatus.String
	}
	if updatedUser.PartnerName.Valid {
		responsePartnerName = &updatedUser.PartnerName.String
	}

	u.log.Info(ctx, "Password changed successfully", zap.String("user-id", userID.String()))

	return &dto.User{
		ID:             updatedUser.ID,
		Name:           updatedUser.Name,
		LastName:       updatedUser.Lastname,
		PhoneNumber:    updatedUser.PhoneNumber,
		Role:           updatedUser.Role,
		Status:         string(updatedUser.Status), // Use actual status from database
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		PartnerName:    responsePartnerName,
		ChildrensName:  updatedUser.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      updatedUser.CreatedAt,
		UpdatedAt:      updatedUser.UpdatedAt,
	}, nil
}

// GetByStatus returns users filtered by status with pagination
func (u *user) GetByStatus(ctx context.Context, status string, page, pageSize int) ([]dto.User, int64, error) {
	// For now, return filtered results from GetAll (temporary implementation)
	// This will be replaced with proper SQL queries after migration
	users, _, err := u.GetAll(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Filter by status (temporary implementation)
	var filteredUsers []dto.User
	for _, user := range users {
		if user.Status == status {
			filteredUsers = append(filteredUsers, user)
		}
	}

	// For count, we'll use a simple approach for now
	filteredTotal := int64(len(filteredUsers))

	u.log.Info(ctx, "retrieved users by status", zap.String("status", status), zap.Int("count", len(filteredUsers)))
	return filteredUsers, filteredTotal, nil
}

// UpdateStatus updates a user's status
func (u *user) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) (*dto.User, error) {
	// Validate status value
	switch status {
	case "PENDING", "ACTIVE", "INACTIVE":
		// Valid status values
	default:
		err := errors.ErrInvalidUserInput.New("invalid status: must be PENDING, ACTIVE, or INACTIVE")
		u.log.Error(ctx, "invalid status for update", zap.String("status", status), zap.String("user-id", userID.String()))
		return nil, err
	}

	// Update user status in database
	updatedUser, err := u.db.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
		Status: status,
		ID:     userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.ErrNoRecordFound.Wrap(err, "user not found")
			u.log.Info(ctx, "User not found for status update", zap.String("user-id", userID.String()))
			return nil, err
		}
		err = errors.ErrWriteError.Wrap(err, "could not update user status")
		u.log.Error(ctx, "unable to update user status", zap.Error(err), zap.String("user-id", userID.String()))
		return nil, err
	}

	// Convert optional fields for response
	var responseTelegramID, responseJobTitle, responseEducation, responseMarriageStatus, responsePartnerName *string
	if updatedUser.TelegramID.Valid {
		responseTelegramID = &updatedUser.TelegramID.String
	}
	if updatedUser.JobTitle.Valid {
		responseJobTitle = &updatedUser.JobTitle.String
	}
	if updatedUser.Education.Valid {
		responseEducation = &updatedUser.Education.String
	}
	if updatedUser.MarriageStatus.Valid {
		responseMarriageStatus = &updatedUser.MarriageStatus.String
	}
	if updatedUser.PartnerName.Valid {
		responsePartnerName = &updatedUser.PartnerName.String
	}

	u.log.Info(ctx, "user status updated successfully", 
		zap.String("user-id", userID.String()),
		zap.String("new-status", status))

	return &dto.User{
		ID:             updatedUser.ID,
		Name:           updatedUser.Name,
		LastName:       updatedUser.Lastname,
		PhoneNumber:    updatedUser.PhoneNumber,
		Role:           updatedUser.Role,
		Status:         string(updatedUser.Status), // Use actual status from database
		JobTitle:       responseJobTitle,
		Education:      responseEducation,
		MarriageStatus: responseMarriageStatus,
		PartnerName:    responsePartnerName,
		ChildrensName:  updatedUser.ChildrensName,
		TelegramID:     responseTelegramID,
		CreatedAt:      updatedUser.CreatedAt,
		UpdatedAt:      updatedUser.UpdatedAt,
	}, nil
}
