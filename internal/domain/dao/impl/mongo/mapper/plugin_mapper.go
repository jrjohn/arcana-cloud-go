package mapper

import (
	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao/impl/mongo/document"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// PluginMapper converts between Plugin entity and PluginDocument.
type PluginMapper struct{}

// NewPluginMapper creates a new PluginMapper instance.
func NewPluginMapper() *PluginMapper {
	return &PluginMapper{}
}

// ToDocument converts a Plugin entity to a PluginDocument.
func (m *PluginMapper) ToDocument(plugin *entity.Plugin) *document.PluginDocument {
	if plugin == nil {
		return nil
	}

	doc := &document.PluginDocument{
		NumericID:   plugin.ID,
		Key:         plugin.Key,
		Name:        plugin.Name,
		Description: plugin.Description,
		Version:     plugin.Version,
		Author:      plugin.Author,
		Type:        string(plugin.Type),
		State:       string(plugin.State),
		Config:      plugin.Config,
		Checksum:    plugin.Checksum,
		Path:        plugin.Path,
		InstalledAt: plugin.InstalledAt,
		EnabledAt:   plugin.EnabledAt,
		CreatedAt:   plugin.CreatedAt,
		UpdatedAt:   plugin.UpdatedAt,
	}

	if plugin.DeletedAt.Valid {
		doc.DeletedAt = &plugin.DeletedAt.Time
	}

	return doc
}

// ToEntity converts a PluginDocument to a Plugin entity.
func (m *PluginMapper) ToEntity(doc *document.PluginDocument) *entity.Plugin {
	if doc == nil {
		return nil
	}

	plugin := &entity.Plugin{
		ID:          doc.NumericID,
		Key:         doc.Key,
		Name:        doc.Name,
		Description: doc.Description,
		Version:     doc.Version,
		Author:      doc.Author,
		Type:        entity.PluginType(doc.Type),
		State:       entity.PluginState(doc.State),
		Config:      doc.Config,
		Checksum:    doc.Checksum,
		Path:        doc.Path,
		InstalledAt: doc.InstalledAt,
		EnabledAt:   doc.EnabledAt,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}

	if doc.DeletedAt != nil {
		plugin.DeletedAt = gorm.DeletedAt{Time: *doc.DeletedAt, Valid: true}
	}

	return plugin
}

// ToEntities converts a slice of PluginDocument to a slice of Plugin entities.
func (m *PluginMapper) ToEntities(docs []*document.PluginDocument) []*entity.Plugin {
	if docs == nil {
		return nil
	}

	plugins := make([]*entity.Plugin, len(docs))
	for i, doc := range docs {
		plugins[i] = m.ToEntity(doc)
	}
	return plugins
}

// ToDocuments converts a slice of Plugin entities to a slice of PluginDocument.
func (m *PluginMapper) ToDocuments(plugins []*entity.Plugin) []*document.PluginDocument {
	if plugins == nil {
		return nil
	}

	docs := make([]*document.PluginDocument, len(plugins))
	for i, plugin := range plugins {
		docs[i] = m.ToDocument(plugin)
	}
	return docs
}
