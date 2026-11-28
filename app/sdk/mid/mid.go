// Package mid provides app level middleware support.
package mid

import (
	"context"
	"errors"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
)

func checkIsError(e web.Encoder) error {
	err, hasError := e.(error)
	if hasError {
		return err
	}

	return nil
}

// =============================================================================

type ctxKey int

const (
	claimKey ctxKey = iota + 1
	userIDKey
	userKey
	projectKey
	runKey
)

func setClaims(ctx context.Context, claims auth.Claims) context.Context {
	return context.WithValue(ctx, claimKey, claims)
}

// GetClaims returns the claims from the context.
func GetClaims(ctx context.Context) auth.Claims {
	v, ok := ctx.Value(claimKey).(auth.Claims)
	if !ok {
		return auth.Claims{}
	}
	return v
}

// GetSubjectID returns the subject id from the claims.
func GetSubjectID(ctx context.Context) uuid.UUID {
	v := GetClaims(ctx)

	subjectID, err := uuid.Parse(v.Subject)
	if err != nil {
		return uuid.UUID{}
	}

	return subjectID
}

func setUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID returns the user id from the context.
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("user id not found in context")
	}

	return v, nil
}

func setUser(ctx context.Context, usr userbus.User) context.Context {
	return context.WithValue(ctx, userKey, usr)
}

// GetUser returns the user from the context.
func GetUser(ctx context.Context) (userbus.User, error) {
	v, ok := ctx.Value(userKey).(userbus.User)
	if !ok {
		return userbus.User{}, errors.New("user not found in context")
	}

	return v, nil
}

func setProject(ctx context.Context, prj projectbus.Project) context.Context {
	return context.WithValue(ctx, projectKey, prj)
}

// GetProject returns the project from the context.
func GetProject(ctx context.Context) (projectbus.Project, error) {
	v, ok := ctx.Value(projectKey).(projectbus.Project)
	if !ok {
		return projectbus.Project{}, errors.New("project not found in context")
	}

	return v, nil
}

func setRun(ctx context.Context, run runbus.Run) context.Context {
	return context.WithValue(ctx, runKey, run)
}

// GetRun returns the run from the context.
func GetRun(ctx context.Context) (runbus.Run, error) {
	v, ok := ctx.Value(runKey).(runbus.Run)
	if !ok {
		return runbus.Run{}, errors.New("run not found in context")
	}

	return v, nil
}
