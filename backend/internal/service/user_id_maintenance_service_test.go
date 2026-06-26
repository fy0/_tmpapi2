//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestUserIDMaintenanceServiceChangeUserIDValidatesBeforeDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	svc := NewUserIDMaintenanceService(db, nil, nil)

	tests := []struct {
		name   string
		req    ChangeUserIDRequest
		reason string
		code   int
	}{
		{
			name: "confirmation required",
			req: ChangeUserIDRequest{
				OldUserID:    1,
				NewUserID:    2,
				Confirmation: "wrong",
			},
			reason: "USER_ID_CHANGE_CONFIRMATION_REQUIRED",
			code:   http.StatusBadRequest,
		},
		{
			name: "ids must be positive",
			req: ChangeUserIDRequest{
				OldUserID:    0,
				NewUserID:    2,
				Confirmation: userIDMaintenanceConfirmation,
			},
			reason: "USER_ID_INVALID_ID",
			code:   http.StatusBadRequest,
		},
		{
			name: "ids must differ",
			req: ChangeUserIDRequest{
				OldUserID:    2,
				NewUserID:    2,
				Confirmation: userIDMaintenanceConfirmation,
			},
			reason: "USER_ID_UNCHANGED",
			code:   http.StatusBadRequest,
		},
		{
			name: "operator cannot change self",
			req: ChangeUserIDRequest{
				OldUserID:      7,
				NewUserID:      8,
				OperatorUserID: 7,
				Confirmation:   userIDMaintenanceConfirmation,
			},
			reason: "USER_ID_SELF_CHANGE_FORBIDDEN",
			code:   http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ChangeUserID(context.Background(), tt.req)
			require.Error(t, err)
			require.Equal(t, tt.code, infraerrors.Code(err))
			require.Equal(t, tt.reason, infraerrors.Reason(err))
		})
	}

}

func TestUserIDMaintenanceServiceSetNextUserIDRejectsNonPositiveBeforeDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	svc := NewUserIDMaintenanceService(db, nil, nil)

	_, err = svc.SetNextUserID(context.Background(), 0)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "USER_ID_INVALID_NEXT_ID", infraerrors.Reason(err))
}
