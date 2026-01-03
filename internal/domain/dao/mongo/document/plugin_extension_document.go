package document

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PluginExtensionDocument represents a plugin extension in MongoDB.
type PluginExtensionDocument struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	NumericID uint               `bson:"numeric_id"` // For compatibility with SQL-based IDs
	PluginID  uint               `bson:"plugin_id"`  // References PluginDocument.NumericID
	Name      string             `bson:"name"`
	Type      string             `bson:"type"`
	Path      string             `bson:"path,omitempty"`
	Handler   string             `bson:"handler,omitempty"`
	Config    string             `bson:"config,omitempty"`
	CreatedAt time.Time          `bson:"created_at"`
	DeletedAt *time.Time         `bson:"deleted_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for plugin extensions.
func (PluginExtensionDocument) CollectionName() string {
	return "plugin_extensions"
}

// IsDeleted returns true if the document has been soft-deleted.
func (d *PluginExtensionDocument) IsDeleted() bool {
	return d.DeletedAt != nil
}
