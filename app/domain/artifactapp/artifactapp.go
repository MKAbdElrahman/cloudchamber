// Package artifactapp maintains the app layer API for reading stored files.
package artifactapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
)

type app struct {
	log         *logger.Logger
	artifactBus artifactbus.ExtBusiness
}

func newApp(log *logger.Logger, artifactBus artifactbus.ExtBusiness) *app {
	return &app{
		log:         log,
		artifactBus: artifactBus,
	}
}

func (a *app) readURL(ctx context.Context, r *http.Request) web.Encoder {
	ownerID, err := mid.GetUserID(ctx)
	if err != nil {
		return errs.New(errs.Unauthenticated, err)
	}

	fileID, fieldErr := toBusFileID(web.Param(r, "file_id"))
	if fieldErr != nil {
		return fieldErr
	}

	playback, err := a.artifactBus.ReadURL(ctx, ownerID, fileID)
	if err != nil {
		if errors.Is(err, artifactbus.ErrNotFound) {
			return errs.New(errs.NotFound, errors.New("file not found"))
		}
		a.log.Error(ctx, "artifact.read-url", "err", err, "file_id", fileID)
		return errs.Errorf(errs.Internal, "read url: %s", err)
	}

	return fromBusPlaybackResponse(playback)
}

func toBusFileID(value string) (uuid.UUID, *errs.Error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, errs.NewFieldErrors("fileId", errors.New("fileId is required"))
	}
	return id, nil
}
