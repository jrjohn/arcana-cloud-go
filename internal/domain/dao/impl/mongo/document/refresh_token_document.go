package document

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RefreshTokenDocument represents a refresh token in MongoDB.
type RefreshTokenDocument struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	NumericID uint               `bson:"numeric_id"` // For compatibility with SQL-based IDs
	UserID    uint               `bson:"user_id"`    // References UserDocument.NumericID
	Token     string             `bson:"token"`
	ExpiresAt time.Time          `bson:"expires_at"`
	Revoked   bool               `bson:"revoked"`
	CreatedAt time.Time          `bson:"created_at"`
	DeletedAt *time.Time         `bson:"deleted_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for refresh tokens.
func (RefreshTokenDocument) CollectionName() string {
	return "refresh_tokens"
}

// IsExpired returns true if the token has expired.
func (d *RefreshTokenDocument) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// IsValid returns true if the token is not revoked and not expired.
func (d *RefreshTokenDocument) IsValid() bool {
	return !d.Revoked && !d.IsExpired()
}

// IsDeleted returns true if the document has been soft-deleted.
func (d *RefreshTokenDocument) IsDeleted() bool {
	return d.DeletedAt != nil
}
