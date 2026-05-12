package socialpublish

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerifyWebhookSignature(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	payload := []byte(`{"id":"evt_1","type":"post.published","data":{}}`)
	signature := SignWebhookPayload("whsec_test", payload, now)

	err := VerifyWebhookSignature("whsec_test", signature, payload, now, time.Minute)
	require.NoError(t, err)

	err = VerifyWebhookSignature("whsec_test", signature, []byte(`{}`), now, time.Minute)
	require.ErrorContains(t, err, "signature mismatch")
}

func TestWebhookRouterDispatchesVerifiedEvent(t *testing.T) {
	t.Parallel()

	now := time.Now()
	payload := []byte(`{"id":"evt_1","type":"post.published","data":{"post_id":"post_1"}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set(signatureHeader, SignWebhookPayload("whsec_test", payload, now))

	router := NewWebhookRouter("whsec_test").WithClockTolerance(time.Minute)
	var handled WebhookEvent
	router.Handle("post.published", func(ctx context.Context, event WebhookEvent) error {
		handled = event
		return nil
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "evt_1", handled.ID)
	require.Equal(t, "post.published", handled.Type)
}

func TestWebhookRouterRejectsBadSignature(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{"type":"post.published"}`)))
	req.Header.Set(signatureHeader, "t=1700000000,v1=bad")

	rec := httptest.NewRecorder()
	NewWebhookRouter("whsec_test").ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
