package mapper

import (
	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao/impl/mongo/document"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// RefreshTokenMapper converts between RefreshToken entity and RefreshTokenDocument.
type RefreshTokenMapper struct{}

// NewRefreshTokenMapper creates a new RefreshTokenMapper instance.
func NewRefreshTokenMapper() *RefreshTokenMapper {
	return &RefreshTokenMapper{}
}

// ToDocument converts a RefreshToken entity to a RefreshTokenDocument.
func (m *RefreshTokenMapper) ToDocument(token *entity.RefreshToken) *document.RefreshTokenDocument {
	if token == nil {
		return nil
	}

	doc := &document.RefreshTokenDocument{
		NumericID: token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
		Revoked:   token.Revoked,
		CreatedAt: token.CreatedAt,
	}

	if token.DeletedAt.Valid {
		doc.DeletedAt = &token.DeletedAt.Time
	}

	return doc
}

// ToEntity converts a RefreshTokenDocument to a RefreshToken entity.
// Note: The User relationship is not populated - use a separate query if needed.
func (m *RefreshTokenMapper) ToEntity(doc *document.RefreshTokenDocument) *entity.RefreshToken {
	if doc == nil {
		return nil
	}

	token := &entity.RefreshToken{
		ID:        doc.NumericID,
		UserID:    doc.UserID,
		Token:     doc.Token,
		ExpiresAt: doc.ExpiresAt,
		Revoked:   doc.Revoked,
		CreatedAt: doc.CreatedAt,
	}

	if doc.DeletedAt != nil {
		token.DeletedAt = gorm.DeletedAt{Time: *doc.DeletedAt, Valid: true}
	}

	return token
}

// ToEntities converts a slice of RefreshTokenDocument to a slice of RefreshToken entities.
func (m *RefreshTokenMapper) ToEntities(docs []*document.RefreshTokenDocument) []*entity.RefreshToken {
	if docs == nil {
		return nil
	}

	tokens := make([]*entity.RefreshToken, len(docs))
	for i, doc := range docs {
		tokens[i] = m.ToEntity(doc)
	}
	return tokens
}

// ToDocuments converts a slice of RefreshToken entities to a slice of RefreshTokenDocument.
func (m *RefreshTokenMapper) ToDocuments(tokens []*entity.RefreshToken) []*document.RefreshTokenDocument {
	if tokens == nil {
		return nil
	}

	docs := make([]*document.RefreshTokenDocument, len(tokens))
	for i, token := range tokens {
		docs[i] = m.ToDocument(token)
	}
	return docs
}
