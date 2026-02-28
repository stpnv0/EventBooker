package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/service/ports/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserService_Create_Success(t *testing.T) {
	repo := mocks.NewMockUserRepo(t)
	svc := NewUserService(repo)

	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	chatID := int64(12345)
	input := domain.CreateUserInput{
		Username:       "testuser",
		TelegramChatID: &chatID,
	}

	user, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, &chatID, user.TelegramChatID)
	assert.NotEmpty(t, user.ID)
}

func TestUserService_Create_EmptyUsername(t *testing.T) {
	svc := NewUserService(nil)

	_, err := svc.Create(context.Background(), domain.CreateUserInput{Username: ""})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestUserService_Create_RepoError(t *testing.T) {
	repo := mocks.NewMockUserRepo(t)
	svc := NewUserService(repo)

	repoErr := errors.New("db error")
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(repoErr)

	_, err := svc.Create(context.Background(), domain.CreateUserInput{Username: "user"})

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}

func TestUserService_Create_UsernameTaken(t *testing.T) {
	repo := mocks.NewMockUserRepo(t)
	svc := NewUserService(repo)

	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.ErrUsernameTaken)

	_, err := svc.Create(context.Background(), domain.CreateUserInput{Username: "taken"})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUsernameTaken)
}

func TestUserService_GetByID_Success(t *testing.T) {
	repo := mocks.NewMockUserRepo(t)
	svc := NewUserService(repo)

	expected := &domain.User{ID: "u1", Username: "alice"}
	repo.EXPECT().GetByID(mock.Anything, "u1").Return(expected, nil)

	user, err := svc.GetByID(context.Background(), "u1")

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	repo := mocks.NewMockUserRepo(t)
	svc := NewUserService(repo)

	repo.EXPECT().GetByID(mock.Anything, "missing").Return(nil, domain.ErrUserNotFound)

	_, err := svc.GetByID(context.Background(), "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserService_List_Success(t *testing.T) {
	repo := mocks.NewMockUserRepo(t)
	svc := NewUserService(repo)

	users := []*domain.User{{ID: "u1"}, {ID: "u2"}}
	repo.EXPECT().List(mock.Anything).Return(users, nil)

	result, err := svc.List(context.Background())

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestUserService_List_Error(t *testing.T) {
	repo := mocks.NewMockUserRepo(t)
	svc := NewUserService(repo)

	repo.EXPECT().List(mock.Anything).Return(nil, errors.New("db error"))

	_, err := svc.List(context.Background())

	require.Error(t, err)
}
