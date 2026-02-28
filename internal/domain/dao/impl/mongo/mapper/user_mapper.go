// Package mapper provides conversion functions between domain entities and MongoDB documents.
package mapper

import (
	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao/impl/mongo/document"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// UserMapper converts between User entity and UserDocument.
type UserMapper struct{}

// NewUserMapper creates a new UserMapper instance.
func NewUserMapper() *UserMapper {
	return &UserMapper{}
}

// ToDocument converts a User entity to a UserDocument.
func (m *UserMapper) ToDocument(user *entity.User) *document.UserDocument {
	if user == nil {
		return nil
	}

	doc := &document.UserDocument{
		NumericID:  user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Password:   user.Password,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Role:       string(user.Role),
		IsActive:   user.IsActive,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}

	if user.DeletedAt.Valid {
		doc.DeletedAt = &user.DeletedAt.Time
	}

	return doc
}

// ToEntity converts a UserDocument to a User entity.
func (m *UserMapper) ToEntity(doc *document.UserDocument) *entity.User {
	if doc == nil {
		return nil
	}

	user := &entity.User{
		ID:         doc.NumericID,
		Username:   doc.Username,
		Email:      doc.Email,
		Password:   doc.Password,
		FirstName:  doc.FirstName,
		LastName:   doc.LastName,
		Role:       entity.UserRole(doc.Role),
		IsActive:   doc.IsActive,
		IsVerified: doc.IsVerified,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}

	if doc.DeletedAt != nil {
		user.DeletedAt = gorm.DeletedAt{Time: *doc.DeletedAt, Valid: true}
	}

	return user
}

// ToEntities converts a slice of UserDocument to a slice of User entities.
func (m *UserMapper) ToEntities(docs []*document.UserDocument) []*entity.User {
	if docs == nil {
		return nil
	}

	users := make([]*entity.User, len(docs))
	for i, doc := range docs {
		users[i] = m.ToEntity(doc)
	}
	return users
}

// ToDocuments converts a slice of User entities to a slice of UserDocument.
func (m *UserMapper) ToDocuments(users []*entity.User) []*document.UserDocument {
	if users == nil {
		return nil
	}

	docs := make([]*document.UserDocument, len(users))
	for i, user := range users {
		docs[i] = m.ToDocument(user)
	}
	return docs
}
