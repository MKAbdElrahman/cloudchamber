// Package tusdapp handles internal tusd upload hooks (auth/bind, finalize/record).
package tusdapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type app struct {
	authClient  authclient.Authenticator
	uploadBus   uploadbus.ExtBusiness
	objectStore *objectstore.Client
}

func newApp(authClient authclient.Authenticator, uploadBus uploadbus.ExtBusiness, objectStore *objectstore.Client) *app {
	return &app{
		authClient:  authClient,
		uploadBus:   uploadBus,
		objectStore: objectStore,
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

	case "post-finish":
		if err := a.postFinish(ctx, req); err != nil {
			return errs.New(errs.Internal, err)
		}
		return hookResponse{}

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

	uploadID := fmt.Sprintf("sources/%s/%s", resp.UserID, uuid.NewString())

	return hookResponse{
		ChangeFileInfo: &hookFileInfo{
			ID: uploadID,
		},
	}
}

func (a *app) postFinish(ctx context.Context, req hookRequest) error {
	if a.uploadBus == nil || a.objectStore == nil {
		return fmt.Errorf("upload finalize dependencies not configured")
	}

	objectKey := req.Event.Upload.Storage.Key
	if objectKey == "" {
		objectKey = baseObjectID(req.Event.Upload.ID)
	}
	if objectKey == "" {
		return fmt.Errorf("post-finish: missing object key")
	}

	bucket := req.Event.Upload.Storage.Bucket
	if bucket == "" {
		bucket = a.objectStore.Bucket()
	}

	ownerID, err := parseOwnerID(objectKey)
	if err != nil {
		return fmt.Errorf("post-finish: %w", err)
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
		return fmt.Errorf("post-finish: verify object: %w", err)
	}
	span.End()

	if req.Event.Upload.Size > 0 && info.Size != req.Event.Upload.Size {
		return fmt.Errorf("post-finish: size mismatch: storage=%d hook=%d", info.Size, req.Event.Upload.Size)
	}

	sourceRef := fmt.Sprintf("s3://%s/%s", bucket, objectKey)
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
		SourceRef:    sourceRef,
		SizeBytes:    info.Size,
		ETag:         etag,
		Status:       uploadbus.MustParseStatus(uploadbus.StatusReady),
		TusdUploadID: &tusdID,
	}

	ctx, span = tracer.Start(ctx, "app.domain.tusdapp.record_owned_source_ref",
		trace.WithAttributes(
			attribute.String("object_key", objectKey),
			attribute.String("source_ref", sourceRef),
		),
	)
	if _, err := a.uploadBus.UpsertReady(ctx, nu); err != nil {
		span.End()
		return fmt.Errorf("post-finish: upsert ready: %w", err)
	}
	span.End()

	return nil
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
	if i := strings.Index(uploadID, "+"); i >= 0 {
		return uploadID[:i]
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
