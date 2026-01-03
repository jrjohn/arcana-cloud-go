package document

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PluginDocument represents a plugin in MongoDB.
type PluginDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	NumericID   uint               `bson:"numeric_id"` // For compatibility with SQL-based IDs
	Key         string             `bson:"key"`
	Name        string             `bson:"name"`
	Description string             `bson:"description,omitempty"`
	Version     string             `bson:"version"`
	Author      string             `bson:"author,omitempty"`
	Type        string             `bson:"type"`
	State       string             `bson:"state"`
	Config      string             `bson:"config,omitempty"`
	Checksum    string             `bson:"checksum,omitempty"`
	Path        string             `bson:"path,omitempty"`
	InstalledAt time.Time          `bson:"installed_at"`
	EnabledAt   *time.Time         `bson:"enabled_at,omitempty"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
	DeletedAt   *time.Time         `bson:"deleted_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for plugins.
func (PluginDocument) CollectionName() string {
	return "plugins"
}

// IsEnabled returns true if the plugin state is "ENABLED".
func (d *PluginDocument) IsEnabled() bool {
	return d.State == "ENABLED"
}

// IsDeleted returns true if the document has been soft-deleted.
func (d *PluginDocument) IsDeleted() bool {
	return d.DeletedAt != nil
}
