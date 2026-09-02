package rooms_service_test

import (
	"context"
	"testing"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	rooms_service "go-chat/internal/features/rooms/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Мок репозитория ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateRoom(ctx context.Context, room domain_models.Room) (domain_models.Room, error) {
	args := m.Called(ctx, room)
	return args.Get(0).(domain_models.Room), args.Error(1)
}

func (m *MockRepository) GetRoom(ctx context.Context, roomID string) (domain_models.Room, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).(domain_models.Room), args.Error(1)
}

func (m *MockRepository) GetPublicRooms(ctx context.Context, limit, offset int) ([]domain_models.Room, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]domain_models.Room), args.Error(1)
}

func (m *MockRepository) GetUserRooms(ctx context.Context, userID string) ([]domain_models.Room, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain_models.Room), args.Error(1)
}

func (m *MockRepository) DeleteRoom(ctx context.Context, roomID, ownerID string) error {
	args := m.Called(ctx, roomID, ownerID)
	return args.Error(0)
}

func (m *MockRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetMember(ctx context.Context, roomID, userID string) (domain_models.RoomMember, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Get(0).(domain_models.RoomMember), args.Error(1)
}

func (m *MockRepository) GetMembers(ctx context.Context, roomID string) ([]domain_models.RoomMember, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).([]domain_models.RoomMember), args.Error(1)
}

func (m *MockRepository) AddMember(ctx context.Context, roomID, userID string, role domain_models.MemberRole) error {
	args := m.Called(ctx, roomID, userID, role)
	return args.Error(0)
}

func (m *MockRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	args := m.Called(ctx, roomID, userID)
	return args.Error(0)
}

func (m *MockRepository) UpdateMemberRole(ctx context.Context, roomID, userID string, role domain_models.MemberRole) error {
	args := m.Called(ctx, roomID, userID, role)
	return args.Error(0)
}

func (m *MockRepository) MuteMember(ctx context.Context, roomID, userID string, mutedUntil time.Time) error {
	args := m.Called(ctx, roomID, userID, mutedUntil)
	return args.Error(0)
}

func (m *MockRepository) UnmuteMember(ctx context.Context, roomID, userID string) error {
	args := m.Called(ctx, roomID, userID)
	return args.Error(0)
}

func (m *MockRepository) CreateInvite(ctx context.Context, invite domain_models.RoomInvite) (domain_models.RoomInvite, error) {
	args := m.Called(ctx, invite)
	return args.Get(0).(domain_models.RoomInvite), args.Error(1)
}

func (m *MockRepository) GetInviteByToken(ctx context.Context, token string) (domain_models.RoomInvite, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(domain_models.RoomInvite), args.Error(1)
}

func (m *MockRepository) TryIncrementInviteUses(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRepository) AcceptInvite(ctx context.Context, token, userID string) (domain_models.Room, domain_models.RoomInvite, error) {
	args := m.Called(ctx, token, userID)
	return args.Get(0).(domain_models.Room), args.Get(1).(domain_models.RoomInvite), args.Error(2)
}

func (m *MockRepository) DeactivateInvite(ctx context.Context, token, userID string) error {
	args := m.Called(ctx, token, userID)
	return args.Error(0)
}

func (m *MockRepository) GetRoomInvites(ctx context.Context, roomID string) ([]domain_models.RoomInvite, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).([]domain_models.RoomInvite), args.Error(1)
}

func (m *MockRepository) CreateBan(ctx context.Context, ban domain_models.RoomBan) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockRepository) GetBan(ctx context.Context, roomID, userID string) (domain_models.RoomBan, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Get(0).(domain_models.RoomBan), args.Error(1)
}

func (m *MockRepository) RemoveBan(ctx context.Context, roomID, userID string) error {
	args := m.Called(ctx, roomID, userID)
	return args.Error(0)
}

func (m *MockRepository) GetRoomBans(ctx context.Context, roomID string) ([]domain_models.RoomBan, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).([]domain_models.RoomBan), args.Error(1)
}

func (m *MockRepository) IsBanned(ctx context.Context, roomID, userID string) (bool, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) CleanExpiredBans(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// --- Хелперы ---

const (
	ownerID  = "owner-user-id"
	memberID = "member-user-id"
	adminID  = "admin-user-id"
	roomID   = "room-id"
)

func newRoom(isPrivate bool) domain_models.Room {
	return domain_models.NewRoom(roomID, "general", "test", isPrivate, false, ownerID, time.Now())
}

func newMember(userID string, role domain_models.MemberRole) domain_models.RoomMember {
	return domain_models.NewRoomMember(roomID, userID, "username", role, time.Now(), nil)
}

func newInvite(uses, maxUses int, isActive bool, expiresAt *time.Time) domain_models.RoomInvite {
	return domain_models.NewRoomInvite("invite-id", roomID, "token123", ownerID, maxUses, uses, expiresAt, time.Now())
}

func newService() (*rooms_service.RoomsService, *MockRepository) {
	repo := new(MockRepository)
	svc := rooms_service.NewRoomsService(repo)
	return svc, repo
}

// --- CreateRoom тесты ---

func TestCreateRoom_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)

	repo.On("CreateRoom", ctx, mock.Anything).Return(room, nil)

	result, err := svc.CreateRoom(ctx, "general", "test", ownerID, false)

	assert.NoError(t, err)
	assert.Equal(t, roomID, result.ID)
	repo.AssertExpectations(t)
}

func TestCreateRoom_EmptyName(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	_, err := svc.CreateRoom(ctx, "", "test", ownerID, false)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

// --- DeleteRoom тесты ---

func TestDeleteRoom_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)

	repo.On("GetRoom", ctx, roomID).Return(room, nil)
	repo.On("DeleteRoom", ctx, roomID, ownerID).Return(nil)

	err := svc.DeleteRoom(ctx, roomID, ownerID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteRoom_NotOwner(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)

	repo.On("GetRoom", ctx, roomID).Return(room, nil)

	err := svc.DeleteRoom(ctx, roomID, memberID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "DeleteRoom")
}

// --- JoinPublicRoom тесты ---

func TestJoinPublicRoom_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)

	repo.On("GetRoom", ctx, roomID).Return(room, nil)
	repo.On("IsBanned", ctx, roomID, memberID).Return(false, nil)
	repo.On("IsMember", ctx, roomID, memberID).Return(false, nil)
	repo.On("AddMember", ctx, roomID, memberID, domain_models.MemberRoleMember).Return(nil)

	err := svc.JoinPublicRoom(ctx, roomID, memberID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestJoinPublicRoom_PrivateRoom(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(true)

	repo.On("GetRoom", ctx, roomID).Return(room, nil)

	err := svc.JoinPublicRoom(ctx, roomID, memberID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "AddMember")
}

func TestJoinPublicRoom_AlreadyMember(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)

	repo.On("GetRoom", ctx, roomID).Return(room, nil)
	repo.On("IsBanned", ctx, roomID, memberID).Return(false, nil)
	repo.On("IsMember", ctx, roomID, memberID).Return(true, nil)

	err := svc.JoinPublicRoom(ctx, roomID, memberID)

	assert.ErrorIs(t, err, core_error.ErrConflict)
	repo.AssertNotCalled(t, "AddMember")
}

// --- LeaveRoom тесты ---

func TestLeaveRoom_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)

	repo.On("GetRoom", ctx, roomID).Return(room, nil)
	repo.On("RemoveMember", ctx, roomID, memberID).Return(nil)

	err := svc.LeaveRoom(ctx, roomID, memberID)

	assert.NoError(t, err)
}

func TestLeaveRoom_OwnerCannotLeave(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)

	repo.On("GetRoom", ctx, roomID).Return(room, nil)

	err := svc.LeaveRoom(ctx, roomID, ownerID)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
	repo.AssertNotCalled(t, "RemoveMember")
}

// --- KickMember тесты ---

func TestKickMember_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)
	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)
	repo.On("RemoveMember", ctx, roomID, memberID).Return(nil)

	err := svc.KickMember(ctx, roomID, ownerID, memberID)

	assert.NoError(t, err)
}

func TestKickMember_CannotKickOwner(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, adminID).Return(newMember(adminID, domain_models.MemberRoleAdmin), nil)
	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)

	err := svc.KickMember(ctx, roomID, adminID, ownerID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "RemoveMember")
}

func TestKickMember_AdminCannotKickAdmin(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	otherAdminID := "other-admin-id"

	repo.On("GetMember", ctx, roomID, adminID).Return(newMember(adminID, domain_models.MemberRoleAdmin), nil)
	repo.On("GetMember", ctx, roomID, otherAdminID).Return(newMember(otherAdminID, domain_models.MemberRoleAdmin), nil)

	err := svc.KickMember(ctx, roomID, adminID, otherAdminID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
}

func TestKickMember_NotAdmin(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)

	err := svc.KickMember(ctx, roomID, memberID, ownerID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
}

func TestKickMember_Self(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	err := svc.KickMember(ctx, roomID, ownerID, ownerID)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

// --- AcceptInvite тесты ---

func TestAcceptInvite_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	room := newRoom(false)
	invite := newInvite(0, 1, true, nil)

	repo.On("AcceptInvite", ctx, "token123", memberID).Return(room, invite, nil)
	repo.On("IsBanned", ctx, roomID, memberID).Return(false, nil)

	result, _, err := svc.AcceptInvite(ctx, "token123", memberID)

	assert.NoError(t, err)
	assert.Equal(t, roomID, result.ID)
}

func TestAcceptInvite_AlreadyMember(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("AcceptInvite", ctx, "token123", memberID).Return(domain_models.Room{}, domain_models.RoomInvite{}, core_error.ErrConflict)

	_, _, err := svc.AcceptInvite(ctx, "token123", memberID)

	assert.ErrorIs(t, err, core_error.ErrConflict)
}

func TestAcceptInvite_Expired(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("AcceptInvite", ctx, "token123", memberID).Return(domain_models.Room{}, domain_models.RoomInvite{}, assert.AnError)

	_, _, err := svc.AcceptInvite(ctx, "token123", memberID)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

func TestAcceptInvite_Inactive(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	repo.On("AcceptInvite", ctx, "token123", memberID).Return(domain_models.Room{}, domain_models.RoomInvite{}, assert.AnError)

	_, _, err := svc.AcceptInvite(ctx, "token123", memberID)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

// --- UpdateMemberRole тесты ---

func TestUpdateMemberRole_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)
	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)
	repo.On("UpdateMemberRole", ctx, roomID, memberID, domain_models.MemberRoleAdmin).Return(nil)

	err := svc.UpdateMemberRole(ctx, roomID, ownerID, memberID, domain_models.MemberRoleAdmin)

	assert.NoError(t, err)
}

func TestUpdateMemberRole_CannotChangeOwnerRole(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)
	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)

	err := svc.UpdateMemberRole(ctx, roomID, ownerID, ownerID, domain_models.MemberRoleAdmin)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
	repo.AssertNotCalled(t, "UpdateMemberRole")
}

func TestUpdateMemberRole_CannotAssignOwner(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)
	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)

	err := svc.UpdateMemberRole(ctx, roomID, ownerID, memberID, domain_models.MemberRoleOwner)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
	repo.AssertNotCalled(t, "UpdateMemberRole")
}

func TestUpdateMemberRole_NotOwner(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)

	err := svc.UpdateMemberRole(ctx, roomID, memberID, adminID, domain_models.MemberRoleAdmin)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "UpdateMemberRole")
}

// --- MuteMember тесты ---

func TestMuteMember_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	mutedUntil := time.Now().Add(1 * time.Hour)

	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)
	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)
	repo.On("MuteMember", ctx, roomID, memberID, mutedUntil).Return(nil)

	err := svc.MuteMember(ctx, roomID, ownerID, memberID, mutedUntil)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMuteMember_CannotMuteSelf(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()
	mutedUntil := time.Now().Add(1 * time.Hour)

	err := svc.MuteMember(ctx, roomID, ownerID, ownerID, mutedUntil)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

func TestMuteMember_CannotMuteOwner(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	mutedUntil := time.Now().Add(1 * time.Hour)

	repo.On("GetMember", ctx, roomID, adminID).Return(newMember(adminID, domain_models.MemberRoleAdmin), nil)
	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)

	err := svc.MuteMember(ctx, roomID, adminID, ownerID, mutedUntil)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "MuteMember")
}

func TestMuteMember_AdminCannotMuteAdmin(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	mutedUntil := time.Now().Add(1 * time.Hour)
	otherAdminID := "other-admin-id"

	repo.On("GetMember", ctx, roomID, adminID).Return(newMember(adminID, domain_models.MemberRoleAdmin), nil)
	repo.On("GetMember", ctx, roomID, otherAdminID).Return(newMember(otherAdminID, domain_models.MemberRoleAdmin), nil)

	err := svc.MuteMember(ctx, roomID, adminID, otherAdminID, mutedUntil)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "MuteMember")
}

func TestMuteMember_NotAdmin(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	mutedUntil := time.Now().Add(1 * time.Hour)

	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)

	err := svc.MuteMember(ctx, roomID, memberID, ownerID, mutedUntil)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "MuteMember")
}

// --- UnmuteMember тесты ---

func TestUnmuteMember_Success(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)
	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)
	repo.On("UnmuteMember", ctx, roomID, memberID).Return(nil)

	err := svc.UnmuteMember(ctx, roomID, ownerID, memberID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUnmuteMember_CannotUnmuteSelf(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	err := svc.UnmuteMember(ctx, roomID, ownerID, ownerID)

	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

func TestUnmuteMember_CannotUnmuteOwner(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, adminID).Return(newMember(adminID, domain_models.MemberRoleAdmin), nil)
	repo.On("GetMember", ctx, roomID, ownerID).Return(newMember(ownerID, domain_models.MemberRoleOwner), nil)

	err := svc.UnmuteMember(ctx, roomID, adminID, ownerID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "UnmuteMember")
}

func TestUnmuteMember_AdminCannotUnmuteAdmin(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()
	otherAdminID := "other-admin-id"

	repo.On("GetMember", ctx, roomID, adminID).Return(newMember(adminID, domain_models.MemberRoleAdmin), nil)
	repo.On("GetMember", ctx, roomID, otherAdminID).Return(newMember(otherAdminID, domain_models.MemberRoleAdmin), nil)

	err := svc.UnmuteMember(ctx, roomID, adminID, otherAdminID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "UnmuteMember")
}

func TestUnmuteMember_NotAdmin(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	repo.On("GetMember", ctx, roomID, memberID).Return(newMember(memberID, domain_models.MemberRoleMember), nil)

	err := svc.UnmuteMember(ctx, roomID, memberID, ownerID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	repo.AssertNotCalled(t, "UnmuteMember")
}
