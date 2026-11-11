// Package tusdapp handles internal tusd upload hooks (auth/bind, finalize/record).
package tusdapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"errors"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type app struct {
	authClient  authclient.Authenticator
	uploadBus   uploadbus.ExtBusiness
	artifactBus artifactbus.ExtBusiness
	projectBus  projectbus.ExtBusiness
	objectStore *objectstore.Client
	db          sqldb.Beginner
}

func newApp(authClient authclient.Authenticator, uploadBus uploadbus.ExtBusiness, artifactBus artifactbus.ExtBusiness, projectBus projectbus.ExtBusiness, objectStore *objectstore.Client, db sqldb.Beginner) *app {
	return &app{
		authClient:  authClient,
		uploadBus:   uploadBus,
		artifactBus: artifactBus,
		projectBus:  projectBus,
		objectStore: objectStore,
		db:          db,
	}
}

func (a *app) handle(ctx context.Context, r *http.Request) web.Encoder {
	var req hookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return hookResponse{
			RejectUpload: true,
			HTTPResponse: &hookHTTPResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"message":"invalid hook request"}`,
				Header:     map[string]string{"Content-Type": "application/json"},
			},
		}
	}

	switch req.Type {
	case "pre-create":
		return a.preCreate(ctx, req)

	case "pre-finish":
		return a.preFinish(ctx, req)

	case "pre-terminate":
		return a.preTerminate(ctx, req)

	case "post-terminate":
		if err := a.postTerminate(ctx, req); err != nil {
			return errs.New(errs.Internal, err)
		}
		return hookResponse{}

	default:
		return hookResponse{}
	}
}

func (a *app) preCreate(ctx context.Context, req hookRequest) web.Encoder {
	authorization := headerValue(req.Event.HTTPRequest.Header, "Authorization")
	if authorization == "" {
		return reject(http.StatusUnauthorized, "missing authorization")
	}

	resp, err := a.authClient.Authenticate(ctx, authorization)
	if err != nil {
		return reject(http.StatusUnauthorized, "authentication failed")
	}

	projectID, err := metadataProjectID(req)
	if err != nil {
		return reject(http.StatusBadRequest, "projectId is required")
	}
	prj, err := a.projectBus.QueryByID(ctx, projectID)
	if err != nil || prj.OwnerID != resp.UserID {
		return reject(http.StatusForbidden, "project not found")
	}

	uploadID := fmt.Sprintf("sources/%s/%s", resp.UserID, uuid.NewString())

	return hookResponse{
		ChangeFileInfo: &hookFileInfo{
			ID: uploadID,
		},
	}
}

func (a *app) preFinish(ctx context.Context, req hookRequest) web.Encoder {
	if a.uploadBus == nil || a.objectStore == nil || a.projectBus == nil {
		return errs.New(errs.Internal, fmt.Errorf("upload finalize dependencies not configured"))
	}

	objectKey := req.Event.Upload.Storage.Key
	if objectKey == "" {
		objectKey = baseObjectID(req.Event.Upload.ID)
	}
	if objectKey == "" {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: missing object key"))
	}

	bucket := req.Event.Upload.Storage.Bucket
	if bucket == "" {
		bucket = a.objectStore.Bucket()
	}

	ownerID, err := parseOwnerID(objectKey)
	if err != nil {
		return finishError(http.StatusBadRequest, "unexpected object key")
	}

	projectID, err := metadataProjectID(req)
	if err != nil {
		return finishError(http.StatusBadRequest, "projectId is required")
	}
	prj, err := a.projectBus.QueryByID(ctx, projectID)
	switch {
	case err == nil && prj.OwnerID == ownerID:
	case err != nil && errors.Is(err, projectbus.ErrNotFound), err == nil && prj.OwnerID != ownerID:
		return finishError(http.StatusForbidden, "project not found")
	default:
		return errs.New(errs.Internal, err)
	}

	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("app.domain.tusdapp")
	ctx, span := tracer.Start(ctx, "app.domain.tusdapp.verify_finalized_upload",
		trace.WithAttributes(
			attribute.String("object_key", objectKey),
			attribute.String("bucket", bucket),
		),
	)
	info, err := a.objectStore.HeadObject(ctx, bucket, objectKey)
	if err != nil {
		span.End()
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: verify object: %w", err))
	}
	span.End()

	if req.Event.Upload.Size > 0 && info.Size != req.Event.Upload.Size {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: size mismatch: storage=%d hook=%d", info.Size, req.Event.Upload.Size))
	}

	tusdID := req.Event.Upload.ID

	var etag *string
	if info.ETag != "" {
		e := info.ETag
		etag = &e
	}

	nu := uploadbus.NewUpload{
		ObjectKey:    objectKey,
		OwnerID:      ownerID,
		Bucket:       bucket,
		ObjectKeyRef: objectKey,
		SizeBytes:    info.Size,
		ETag:         etag,
		Status:       uploadbus.MustParseStatus(uploadbus.StatusReady),
		TusdUploadID: &tusdID,
	}

	fileID, err := artifactbus.FileIDFromObjectKey(objectKey)
	if err != nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: file id: %w", err))
	}

	contentType := ""
	name := ""
	if req.Event.Upload.MetaData != nil {
		contentType = req.Event.Upload.MetaData["filetype"]
		name = req.Event.Upload.MetaData["filename"]
	}

	ctx, span = tracer.Start(ctx, "app.domain.tusdapp.record_artifact",
		trace.WithAttributes(
			attribute.String("object_key", objectKey),
			attribute.String("file_id", fileID.String()),
		),
	)
	defer span.End()

	if a.db == nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: database not configured"))
	}
	tx, err := a.db.Begin()
	if err != nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: begin: %w", err))
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	uploadBus, err := a.uploadBus.NewWithTx(tx)
	if err != nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: upload tx: %w", err))
	}
	artifactBus, err := a.artifactBus.NewWithTx(tx)
	if err != nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: artifact tx: %w", err))
	}
	if _, err := uploadBus.UpsertReady(ctx, nu); err != nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: upsert ready: %w", err))
	}
	if _, err := artifactBus.Register(ctx, artifactbus.NewArtifact{
		FileID:      fileID,
		OwnerID:     ownerID,
		ProjectID:   projectID,
		Bucket:      bucket,
		ObjectKey:   objectKey,
		Kind:        artifactbus.MustParseKind(artifactbus.KindSource),
		ContentType: contentType,
		Name:        name,
		Status:      artifactbus.MustParseStatus(artifactbus.StatusReady),
	}); err != nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: register artifact: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return errs.New(errs.Internal, fmt.Errorf("pre-finish: commit: %w", err))
	}
	committed = true

	return hookResponse{}
}

func (a *app) preTerminate(ctx context.Context, req hookRequest) web.Encoder {
	objectKey := req.Event.Upload.Storage.Key
	if objectKey == "" {
		objectKey = baseObjectID(req.Event.Upload.ID)
	}
	fileID, err := artifactbus.FileIDFromObjectKey(objectKey)
	if err != nil {
		return hookResponse{}
	}
	_, err = a.artifactBus.QueryByFileID(ctx, fileID)
	switch {
	case err == nil:
		return hookResponse{RejectTermination: true}
	case errors.Is(err, artifactbus.ErrNotFound):
		return hookResponse{}
	default:
		return errs.New(errs.Internal, err)
	}
}

func (a *app) postTerminate(ctx context.Context, req hookRequest) error {
	if a.uploadBus == nil {
		return nil
	}

	objectKey := req.Event.Upload.Storage.Key
	if objectKey == "" {
		objectKey = baseObjectID(req.Event.Upload.ID)
	}
	if objectKey == "" {
		return nil
	}

	if err := a.uploadBus.MarkTerminated(ctx, objectKey); err != nil {
		return fmt.Errorf("post-terminate: %w", err)
	}

	return nil
}

func baseObjectID(uploadID string) string {
	if uploadID == "" {
		return ""
	}
	if before, _, ok := strings.Cut(uploadID, "+"); ok {
		return before
	}
	return uploadID
}

func parseOwnerID(objectKey string) (uuid.UUID, error) {
	// Expected: sources/{owner_id}/{id...}
	parts := strings.Split(objectKey, "/")
	if len(parts) < 3 || parts[0] != "sources" {
		return uuid.Nil, fmt.Errorf("unexpected object key format: %q", objectKey)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse owner id from key: %w", err)
	}
	return id, nil
}

func metadataProjectID(req hookRequest) (uuid.UUID, error) {
	if req.Event.Upload.MetaData == nil {
		return uuid.Nil, fmt.Errorf("missing metadata")
	}
	return uuid.Parse(req.Event.Upload.MetaData["projectId"])
}

func finishError(status int, message string) hookResponse {
	body, _ := json.Marshal(map[string]string{"message": message})
	return hookResponse{
		HTTPResponse: &hookHTTPResponse{
			StatusCode: status,
			Body:       string(body),
			Header:     map[string]string{"Content-Type": "application/json"},
		},
	}
}

func reject(status int, message string) hookResponse {
	body, _ := json.Marshal(map[string]string{"message": message})
	return hookResponse{
		RejectUpload: true,
		HTTPResponse: &hookHTTPResponse{
			StatusCode: status,
			Body:       string(body),
			Header:     map[string]string{"Content-Type": "application/json"},
		},
	}
}

func headerValue(headers map[string][]string, name string) string {
	if headers == nil {
		return ""
	}
	if values := headers[name]; len(values) > 0 {
		return values[0]
	}
	for k, values := range headers {
		if strings.EqualFold(k, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
