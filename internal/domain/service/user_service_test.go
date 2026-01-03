package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil/mocks"
)

func setupUserService(t *testing.T) (UserService, *mocks.MockUserRepository) {
	userRepo := mocks.NewMockUserRepository()
	passwordHasher := security.NewPasswordHasher()
	userService := NewUserService(userRepo, passwordHasher)
	return userService, userRepo
}

func TestNewUserService(t *testing.T) {
	userService, _ := setupUserService(t)
	if userService == nil {
		t.Fatal("NewUserService() returned nil")
	}
}

func TestUserService_GetByID_Success(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hash",
		FirstName: "Test",
		LastName:  "User",
		Role:      entity.RoleUser,
		IsActive:  true,
	}
	userRepo.AddUser(user)

	resp, err := userService.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetByID() returned nil response")
	}
	if resp.Username != "testuser" {
		t.Errorf("GetByID() Username = %v, want testuser", resp.Username)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	userService, _ := setupUserService(t)
	ctx := context.Background()

	_, err := userService.GetByID(ctx, 999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetByID() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_GetByID_Error(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.GetByIDErr = expectedErr

	_, err := userService.GetByID(ctx, 1)
	if !errors.Is(err, expectedErr) {
		t.Errorf("GetByID() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_GetByUsername_Success(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	resp, err := userService.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetByUsername() returned nil response")
	}
	if resp.Username != "testuser" {
		t.Errorf("GetByUsername() Username = %v, want testuser", resp.Username)
	}
}

func TestUserService_GetByUsername_NotFound(t *testing.T) {
	userService, _ := setupUserService(t)
	ctx := context.Background()

	_, err := userService.GetByUsername(ctx, "nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetByUsername() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_GetByUsername_Error(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.GetByUsernameErr = expectedErr

	_, err := userService.GetByUsername(ctx, "testuser")
	if !errors.Is(err, expectedErr) {
		t.Errorf("GetByUsername() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_GetByEmail_Success(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	resp, err := userService.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetByEmail() returned nil response")
	}
	if resp.Email != "test@example.com" {
		t.Errorf("GetByEmail() Email = %v, want test@example.com", resp.Email)
	}
}

func TestUserService_GetByEmail_NotFound(t *testing.T) {
	userService, _ := setupUserService(t)
	ctx := context.Background()

	_, err := userService.GetByEmail(ctx, "nonexistent@example.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetByEmail() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_GetByEmail_Error(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.GetByEmailErr = expectedErr

	_, err := userService.GetByEmail(ctx, "test@example.com")
	if !errors.Is(err, expectedErr) {
		t.Errorf("GetByEmail() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_List_Success(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	// Add multiple users
	for i := 1; i <= 15; i++ {
		userRepo.AddUser(&entity.User{
			Username: "user" + string(rune('0'+i)),
			Email:    "user" + string(rune('0'+i)) + "@example.com",
			Password: "hash",
			IsActive: true,
		})
	}

	resp, err := userService.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if resp == nil {
		t.Fatal("List() returned nil response")
	}
	if len(resp.Items) != 10 {
		t.Errorf("List() Items count = %v, want 10", len(resp.Items))
	}
	if resp.PageInfo.TotalItems != 15 {
		t.Errorf("List() TotalItems = %v, want 15", resp.PageInfo.TotalItems)
	}
}

func TestUserService_List_InvalidPage(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	userRepo.AddUser(&entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	})

	// Test page < 1 gets normalized to 1
	resp, err := userService.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if resp.PageInfo.Page != 1 {
		t.Errorf("List() Page = %v, want 1", resp.PageInfo.Page)
	}
}

func TestUserService_List_InvalidSize(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	userRepo.AddUser(&entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	})

	// Test size < 1 gets normalized to 10
	resp, err := userService.List(ctx, 1, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if resp.PageInfo.Size != 10 {
		t.Errorf("List() PageSize = %v, want 10", resp.PageInfo.Size)
	}

	// Test size > 100 gets normalized to 10
	resp, err = userService.List(ctx, 1, 200)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if resp.PageInfo.Size != 10 {
		t.Errorf("List() PageSize = %v, want 10", resp.PageInfo.Size)
	}
}

func TestUserService_List_Error(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.ListErr = expectedErr

	_, err := userService.List(ctx, 1, 10)
	if !errors.Is(err, expectedErr) {
		t.Errorf("List() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_Update_Success(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hash",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
	}
	userRepo.AddUser(user)

	req := &request.UpdateProfileRequest{
		FirstName: "Updated",
		LastName:  "Name",
	}

	resp, err := userService.Update(ctx, user.ID, req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if resp.FirstName != "Updated" {
		t.Errorf("Update() FirstName = %v, want Updated", resp.FirstName)
	}
	if resp.LastName != "Name" {
		t.Errorf("Update() LastName = %v, want Name", resp.LastName)
	}
}

func TestUserService_Update_EmailChange(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	req := &request.UpdateProfileRequest{
		Email: "newemail@example.com",
	}

	resp, err := userService.Update(ctx, user.ID, req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if resp.Email != "newemail@example.com" {
		t.Errorf("Update() Email = %v, want newemail@example.com", resp.Email)
	}
}

func TestUserService_Update_EmailExists(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	// Add two users
	user1 := &entity.User{
		Username: "user1",
		Email:    "user1@example.com",
		Password: "hash",
		IsActive: true,
	}
	user2 := &entity.User{
		Username: "user2",
		Email:    "user2@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user1)
	userRepo.AddUser(user2)

	// Try to update user1's email to user2's email
	req := &request.UpdateProfileRequest{
		Email: "user2@example.com",
	}

	_, err := userService.Update(ctx, user1.ID, req)
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("Update() error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestUserService_Update_UserNotFound(t *testing.T) {
	userService, _ := setupUserService(t)
	ctx := context.Background()

	req := &request.UpdateProfileRequest{
		FirstName: "Updated",
	}

	_, err := userService.Update(ctx, 999, req)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Update() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_Update_GetByIDError(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.GetByIDErr = expectedErr

	req := &request.UpdateProfileRequest{
		FirstName: "Updated",
	}

	_, err := userService.Update(ctx, 1, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Update() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_Update_EmailCheckError(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	expectedErr := errors.New("email check error")
	userRepo.ExistsByEmailErr = expectedErr

	req := &request.UpdateProfileRequest{
		Email: "newemail@example.com",
	}

	_, err := userService.Update(ctx, user.ID, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Update() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_Update_UpdateError(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	expectedErr := errors.New("update error")
	userRepo.UpdateErr = expectedErr

	req := &request.UpdateProfileRequest{
		FirstName: "Updated",
	}

	_, err := userService.Update(ctx, user.ID, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Update() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	passwordHasher := security.NewPasswordHasher()
	hashedPassword, _ := passwordHasher.Hash("oldpassword")

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		IsActive: true,
	}
	userRepo.AddUser(user)

	req := &request.ChangePasswordRequest{
		OldPassword: "oldpassword",
		NewPassword: "newpassword",
	}

	err := userService.ChangePassword(ctx, user.ID, req)
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
}

func TestUserService_ChangePassword_UserNotFound(t *testing.T) {
	userService, _ := setupUserService(t)
	ctx := context.Background()

	req := &request.ChangePasswordRequest{
		OldPassword: "oldpassword",
		NewPassword: "newpassword",
	}

	err := userService.ChangePassword(ctx, 999, req)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("ChangePassword() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_ChangePassword_WrongOldPassword(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	passwordHasher := security.NewPasswordHasher()
	hashedPassword, _ := passwordHasher.Hash("correctpassword")

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		IsActive: true,
	}
	userRepo.AddUser(user)

	req := &request.ChangePasswordRequest{
		OldPassword: "wrongpassword",
		NewPassword: "newpassword",
	}

	err := userService.ChangePassword(ctx, user.ID, req)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("ChangePassword() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestUserService_ChangePassword_GetByIDError(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.GetByIDErr = expectedErr

	req := &request.ChangePasswordRequest{
		OldPassword: "oldpassword",
		NewPassword: "newpassword",
	}

	err := userService.ChangePassword(ctx, 1, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("ChangePassword() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_ChangePassword_UpdateError(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	passwordHasher := security.NewPasswordHasher()
	hashedPassword, _ := passwordHasher.Hash("oldpassword")

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		IsActive: true,
	}
	userRepo.AddUser(user)

	expectedErr := errors.New("update error")
	userRepo.UpdateErr = expectedErr

	req := &request.ChangePasswordRequest{
		OldPassword: "oldpassword",
		NewPassword: "newpassword",
	}

	err := userService.ChangePassword(ctx, user.ID, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("ChangePassword() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_Delete_Success(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	err := userService.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestUserService_Delete_Error(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("delete error")
	userRepo.DeleteErr = expectedErr

	err := userService.Delete(ctx, 1)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Delete() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_ExistsByUsername_True(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	exists, err := userService.ExistsByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("ExistsByUsername() error = %v", err)
	}
	if !exists {
		t.Error("ExistsByUsername() = false, want true")
	}
}

func TestUserService_ExistsByUsername_False(t *testing.T) {
	userService, _ := setupUserService(t)
	ctx := context.Background()

	exists, err := userService.ExistsByUsername(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("ExistsByUsername() error = %v", err)
	}
	if exists {
		t.Error("ExistsByUsername() = true, want false")
	}
}

func TestUserService_ExistsByUsername_Error(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.ExistsByUsernameErr = expectedErr

	_, err := userService.ExistsByUsername(ctx, "testuser")
	if !errors.Is(err, expectedErr) {
		t.Errorf("ExistsByUsername() error = %v, want %v", err, expectedErr)
	}
}

func TestUserService_ExistsByEmail_True(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	exists, err := userService.ExistsByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("ExistsByEmail() error = %v", err)
	}
	if !exists {
		t.Error("ExistsByEmail() = false, want true")
	}
}

func TestUserService_ExistsByEmail_False(t *testing.T) {
	userService, _ := setupUserService(t)
	ctx := context.Background()

	exists, err := userService.ExistsByEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("ExistsByEmail() error = %v", err)
	}
	if exists {
		t.Error("ExistsByEmail() = true, want false")
	}
}

func TestUserService_ExistsByEmail_Error(t *testing.T) {
	userService, userRepo := setupUserService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.ExistsByEmailErr = expectedErr

	_, err := userService.ExistsByEmail(ctx, "test@example.com")
	if !errors.Is(err, expectedErr) {
		t.Errorf("ExistsByEmail() error = %v, want %v", err, expectedErr)
	}
}
