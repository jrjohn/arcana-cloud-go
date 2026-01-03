// Package document defines MongoDB document structs for persistence.
// These structs are separate from domain entities to allow for MongoDB-specific
// optimizations and to maintain clean separation of concerns.
package document

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserDocument represents a user in MongoDB.
type UserDocument struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	NumericID  uint               `bson:"numeric_id"` // For compatibility with SQL-based IDs
	Username   string             `bson:"username"`
	Email      string             `bson:"email"`
	Password   string             `bson:"password"`
	FirstName  string             `bson:"first_name,omitempty"`
	LastName   string             `bson:"last_name,omitempty"`
	Role       string             `bson:"role"`
	IsActive   bool               `bson:"is_active"`
	IsVerified bool               `bson:"is_verified"`
	CreatedAt  time.Time          `bson:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at"`
	DeletedAt  *time.Time         `bson:"deleted_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for users.
func (UserDocument) CollectionName() string {
	return "users"
}

// IsDeleted returns true if the document has been soft-deleted.
func (d *UserDocument) IsDeleted() bool {
	return d.DeletedAt != nil
}
