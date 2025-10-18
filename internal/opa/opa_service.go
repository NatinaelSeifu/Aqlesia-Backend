package opa

import (
	"aqlesia/internal/constants/model/dto"
	"aqlesia/platform/logger"
	"context"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
)

//go:embed policies/*.rego
var policyFiles embed.FS

// OPAService provides policy evaluation capabilities
type OPAService interface {
	EvaluatePermission(ctx context.Context, user *dto.User, permission string, resource map[string]interface{}) (bool, error)
	EvaluateUserAccess(ctx context.Context, user *dto.User, action string, targetUserID string) (bool, error)
}

type opaService struct {
	log   logger.Logger
	query rego.PreparedEvalQuery
}

// NewOPAService creates a new OPA service with embedded policies
func NewOPAService(log logger.Logger) (OPAService, error) {
	// Read the RBAC policy file
	policyContent, err := policyFiles.ReadFile("policies/rbac.rego")
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	// Create a new Rego query
	ctx := context.Background()
	
	// Prepare the query to evaluate the "allow" rule
	query, err := rego.New(
		rego.Query("data.rbac.allow"),
		rego.Module("rbac.rego", string(policyContent)),
	).PrepareForEval(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to prepare OPA query: %w", err)
	}

	return &opaService{
		log:   log,
		query: query,
	}, nil
}

// NewOPAServiceFromDir creates OPA service from directory (for development/testing)
func NewOPAServiceFromDir(log logger.Logger, policyDir string) (OPAService, error) {
	// Read policy from directory (fallback for development)
	policyPath := filepath.Join(policyDir, "rbac.rego")
	
	ctx := context.Background()
	
	query, err := rego.New(
		rego.Query("data.rbac.allow"),
		rego.Load([]string{policyPath}, nil),
	).PrepareForEval(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to prepare OPA query from directory: %w", err)
	}

	return &opaService{
		log:   log,
		query: query,
	}, nil
}

// EvaluatePermission evaluates if a user has the required permission
func (o *opaService) EvaluatePermission(ctx context.Context, user *dto.User, permission string, resource map[string]interface{}) (bool, error) {
	// Prepare input for OPA evaluation
	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   user.ID.String(),
			"role": user.Role,
		},
		"permission": permission,
	}
	
	// Add resource information if provided
	if resource != nil {
		input["resource"] = resource
	}

	// Evaluate the policy
	results, err := o.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		o.log.Error(ctx, "OPA policy evaluation failed", 
			zap.Error(err), 
			zap.String("user_id", user.ID.String()), 
			zap.String("permission", permission))
		return false, fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Check if the policy allows the action
	if len(results) == 0 {
		o.log.Debug(ctx, "OPA policy evaluation returned no results", 
			zap.String("user_id", user.ID.String()), 
			zap.String("permission", permission))
		return false, nil
	}

	// Get the first result (should be boolean)
	allowed, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		o.log.Error(ctx, "OPA policy evaluation returned non-boolean result", 
			zap.String("user_id", user.ID.String()), 
			zap.String("permission", permission),
			zap.Any("result", results[0].Expressions[0].Value))
		return false, fmt.Errorf("policy evaluation returned unexpected result type")
	}

	o.log.Debug(ctx, "OPA policy evaluation completed", 
		zap.String("user_id", user.ID.String()), 
		zap.String("permission", permission),
		zap.Bool("allowed", allowed))

	return allowed, nil
}

// EvaluateUserAccess evaluates if a user can perform an action on another user
func (o *opaService) EvaluateUserAccess(ctx context.Context, user *dto.User, action string, targetUserID string) (bool, error) {
	// Create resource context for the target user
	resource := map[string]interface{}{
		"user_id": targetUserID,
	}

	// Map actions to permissions
	var permission string
	switch action {
	case "read":
		if user.ID.String() == targetUserID {
			permission = "users:read_own"
		} else {
			permission = "users:read"
		}
	case "update":
		if user.ID.String() == targetUserID {
			permission = "users:write_own"
		} else {
			permission = "users:write"
		}
	case "delete":
		permission = "users:delete"
	case "list":
		permission = "users:list"
	default:
		return false, fmt.Errorf("unknown action: %s", action)
	}

	return o.EvaluatePermission(ctx, user, permission, resource)
}
