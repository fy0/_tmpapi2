package service

import (
	"context"
	"errors"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type supportTicketRepoStub struct {
	createdTicket  *SupportTicket
	createdMessage *SupportTicketMessage
	tickets        map[int64]*SupportTicket
	listFilters    SupportTicketListFilters
	addedMessage   *SupportTicketMessage
	addedStatus    *string
	addedActorID   *int64
	updateInput    UpdateSupportTicketInput
}

func (r *supportTicketRepoStub) CreateWithMessage(_ context.Context, ticket *SupportTicket, message *SupportTicketMessage) error {
	r.createdTicket = ticket
	r.createdMessage = message
	ticket.ID = 10
	ticket.CreatedAt = time.Unix(100, 0)
	ticket.UpdatedAt = time.Unix(100, 0)
	message.ID = 20
	message.TicketID = ticket.ID
	message.CreatedAt = time.Unix(101, 0)
	if r.tickets == nil {
		r.tickets = map[int64]*SupportTicket{}
	}
	cloned := *ticket
	cloned.Messages = []SupportTicketMessage{*message}
	r.tickets[ticket.ID] = &cloned
	return nil
}

func (r *supportTicketRepoStub) AddMessage(_ context.Context, ticketID int64, message *SupportTicketMessage, status *string, actorID *int64) (*SupportTicket, error) {
	r.addedMessage = message
	r.addedStatus = status
	r.addedActorID = actorID
	ticket, ok := r.tickets[ticketID]
	if !ok {
		return nil, ErrSupportTicketNotFound
	}
	out := *ticket
	if status != nil && *status != "" {
		out.Status = *status
	}
	out.LastMessageAt = time.Unix(200, 0)
	out.UpdatedAt = time.Unix(200, 0)
	return &out, nil
}

func (r *supportTicketRepoStub) GetByID(_ context.Context, id int64) (*SupportTicket, error) {
	ticket, ok := r.tickets[id]
	if !ok {
		return nil, ErrSupportTicketNotFound
	}
	out := *ticket
	if ticket.Messages != nil {
		out.Messages = append([]SupportTicketMessage(nil), ticket.Messages...)
	}
	return &out, nil
}

func (r *supportTicketRepoStub) List(_ context.Context, params pagination.PaginationParams, filters SupportTicketListFilters) ([]SupportTicket, *pagination.PaginationResult, error) {
	r.listFilters = filters
	items := make([]SupportTicket, 0, len(r.tickets))
	for _, ticket := range r.tickets {
		items = append(items, *ticket)
	}
	return items, &pagination.PaginationResult{Total: int64(len(items)), Page: params.Page, PageSize: params.Limit(), Pages: 1}, nil
}

func (r *supportTicketRepoStub) Update(_ context.Context, id int64, input UpdateSupportTicketInput) (*SupportTicket, error) {
	r.updateInput = input
	ticket, ok := r.tickets[id]
	if !ok {
		return nil, ErrSupportTicketNotFound
	}
	out := *ticket
	if input.Status != nil {
		out.Status = *input.Status
	}
	if input.Priority != nil {
		out.Priority = *input.Priority
	}
	return &out, nil
}

type supportTicketUserRepoStub struct {
	users map[int64]*User
}

func (r *supportTicketUserRepoStub) Create(context.Context, *User) error {
	panic("unexpected Create call")
}
func (r *supportTicketUserRepoStub) CreateWithEmailAliasGuard(context.Context, *User) error {
	panic("unexpected CreateWithEmailAliasGuard call")
}
func (r *supportTicketUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	out := *user
	return &out, nil
}
func (r *supportTicketUserRepoStub) GetByIDIncludeDeleted(context.Context, int64) (*User, error) {
	panic("unexpected GetByIDIncludeDeleted call")
}
func (r *supportTicketUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail call")
}
func (r *supportTicketUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}
func (r *supportTicketUserRepoStub) Update(context.Context, *User, UserUpdateFields) error {
	panic("unexpected Update call")
}
func (r *supportTicketUserRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (r *supportTicketUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	panic("unexpected GetUserAvatar call")
}
func (r *supportTicketUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}
func (r *supportTicketUserRepoStub) DeleteUserAvatar(context.Context, int64) error {
	panic("unexpected DeleteUserAvatar call")
}
func (r *supportTicketUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (r *supportTicketUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (r *supportTicketUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}
func (r *supportTicketUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}
func (r *supportTicketUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}
func (r *supportTicketUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}
func (r *supportTicketUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}
func (r *supportTicketUserRepoStub) AdjustBalance(context.Context, int64, float64) (BalanceChange, error) {
	panic("unexpected AdjustBalance call")
}
func (r *supportTicketUserRepoStub) SetBalance(context.Context, int64, float64) (BalanceChange, error) {
	panic("unexpected SetBalance call")
}
func (r *supportTicketUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}
func (r *supportTicketUserRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchSetConcurrency call")
}
func (r *supportTicketUserRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchAddConcurrency call")
}
func (r *supportTicketUserRepoStub) BatchUpdateLimits(context.Context, []int64, *int, *int) (int, error) {
	panic("unexpected BatchUpdateLimits call")
}
func (r *supportTicketUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}
func (r *supportTicketUserRepoStub) ExistsByEmailAlias(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmailAlias call")
}
func (r *supportTicketUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}
func (r *supportTicketUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}
func (r *supportTicketUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}
func (r *supportTicketUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}
func (r *supportTicketUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}
func (r *supportTicketUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}
func (r *supportTicketUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}
func (r *supportTicketUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

type supportTicketVisibilityStub struct {
	value string
}

func (v *supportTicketVisibilityStub) GetSupportTicketEntryVisibility(context.Context) string {
	return v.value
}

func newSupportTicketServiceForTest(visibility string) (*SupportTicketService, *supportTicketRepoStub) {
	repo := &supportTicketRepoStub{tickets: map[int64]*SupportTicket{
		1: {
			ID:            1,
			UserID:        100,
			UserEmail:     "user@example.com",
			UserName:      "user",
			Title:         "Existing",
			Category:      SupportTicketCategoryFeedback,
			Status:        SupportTicketStatusOpen,
			Priority:      SupportTicketPriorityNormal,
			LastMessageAt: time.Unix(1, 0),
			CreatedAt:     time.Unix(1, 0),
			UpdatedAt:     time.Unix(1, 0),
		},
	}}
	userRepo := &supportTicketUserRepoStub{users: map[int64]*User{
		100: {ID: 100, Email: " user@example.com ", Username: " user "},
		200: {ID: 200, Email: "admin@example.com", Username: "admin", Role: RoleAdmin},
	}}
	return &SupportTicketService{
		repo:       repo,
		userRepo:   userRepo,
		visibility: &supportTicketVisibilityStub{value: visibility},
	}, repo
}

func TestSupportTicketService_UserVisibilityBlocksUserAPIs(t *testing.T) {
	svc, repo := newSupportTicketServiceForTest(SupportTicketEntryVisibilityAdmin)

	_, err := svc.CreateForUser(context.Background(), &CreateSupportTicketInput{
		UserID:   100,
		Title:    "Need help",
		Category: SupportTicketCategoryFeedback,
		Priority: SupportTicketPriorityNormal,
		Content:  "Please check this",
	})
	require.ErrorIs(t, err, ErrSupportTicketDisabledForUser)
	require.Nil(t, repo.createdTicket)

	_, _, err = svc.ListForUser(context.Background(), 100, pagination.DefaultPagination(), SupportTicketListFilters{})
	require.ErrorIs(t, err, ErrSupportTicketDisabledForUser)

	_, err = svc.GetForUser(context.Background(), 100, 1)
	require.ErrorIs(t, err, ErrSupportTicketDisabledForUser)

	_, err = svc.AddUserMessage(context.Background(), 100, 1, "More detail")
	require.ErrorIs(t, err, ErrSupportTicketDisabledForUser)
}

func TestSupportTicketService_AdminAPIsIgnoreUserEntryVisibility(t *testing.T) {
	svc, repo := newSupportTicketServiceForTest(SupportTicketEntryVisibilityAdmin)

	created, err := svc.CreateAsAdmin(context.Background(), &CreateSupportTicketInput{
		UserID:   100,
		Title:    " Need help ",
		Category: "unknown",
		Priority: "",
		Content:  " Please check this ",
	})
	require.NoError(t, err)
	require.Equal(t, int64(10), created.ID)
	require.Equal(t, "Need help", created.Title)
	require.Equal(t, SupportTicketCategoryFeedback, created.Category)
	require.Equal(t, SupportTicketPriorityNormal, created.Priority)
	require.Equal(t, "user@example.com", repo.createdTicket.UserEmail)
	require.Equal(t, "user", repo.createdMessage.AuthorName)
	require.Equal(t, SupportTicketAuthorRoleUser, repo.createdMessage.AuthorRole)
}

func TestSupportTicketService_UserCanOnlyAccessOwnTickets(t *testing.T) {
	svc, _ := newSupportTicketServiceForTest(SupportTicketEntryVisibilityAll)

	ticket, err := svc.GetForUser(context.Background(), 100, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), ticket.ID)

	_, err = svc.GetForUser(context.Background(), 999, 1)
	require.ErrorIs(t, err, ErrSupportTicketNotFound)
}

func TestSupportTicketService_AddUserMessageReopensTicket(t *testing.T) {
	svc, repo := newSupportTicketServiceForTest(SupportTicketEntryVisibilityAll)
	repo.tickets[1].Status = SupportTicketStatusResolved

	ticket, err := svc.AddUserMessage(context.Background(), 100, 1, " Reopening ")
	require.NoError(t, err)
	require.Equal(t, SupportTicketStatusOpen, ticket.Status)
	require.NotNil(t, repo.addedStatus)
	require.Equal(t, SupportTicketStatusOpen, *repo.addedStatus)
	require.NotNil(t, repo.addedActorID)
	require.Equal(t, int64(100), *repo.addedActorID)
	require.Equal(t, "Reopening", repo.addedMessage.Content)
	require.Equal(t, SupportTicketAuthorRoleUser, repo.addedMessage.AuthorRole)
}

func TestSupportTicketService_ListForUserForcesUserIDFilter(t *testing.T) {
	svc, repo := newSupportTicketServiceForTest(SupportTicketEntryVisibilityAll)

	_, _, err := svc.ListForUser(context.Background(), 100, pagination.DefaultPagination(), SupportTicketListFilters{
		UserID: 999,
		Search: " query ",
	})
	require.NoError(t, err)
	require.Equal(t, int64(100), repo.listFilters.UserID)
	require.Equal(t, "query", repo.listFilters.Search)
}

func TestSupportTicketService_UpdateValidatesActor(t *testing.T) {
	svc, _ := newSupportTicketServiceForTest(SupportTicketEntryVisibilityAll)

	_, err := svc.Update(context.Background(), 1, UpdateSupportTicketInput{})
	require.ErrorIs(t, err, ErrSupportTicketUserRequired)

	status := "bad"
	_, err = svc.Update(context.Background(), 1, UpdateSupportTicketInput{ActorID: 200, Status: &status})
	require.ErrorIs(t, err, ErrSupportTicketStatusInvalid)
}

func TestSupportTicketService_ErrorSentinelsAreApplicationErrors(t *testing.T) {
	require.True(t, errors.Is(ErrSupportTicketDisabledForUser, infraerrors.FromError(ErrSupportTicketDisabledForUser)))
	require.Equal(t, 403, infraerrors.Code(ErrSupportTicketDisabledForUser))
}
