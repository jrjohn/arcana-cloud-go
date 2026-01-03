package mapper

import (
	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao/mongo/document"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// PluginExtensionMapper converts between PluginExtension entity and PluginExtensionDocument.
type PluginExtensionMapper struct{}

// NewPluginExtensionMapper creates a new PluginExtensionMapper instance.
func NewPluginExtensionMapper() *PluginExtensionMapper {
	return &PluginExtensionMapper{}
}

// ToDocument converts a PluginExtension entity to a PluginExtensionDocument.
func (m *PluginExtensionMapper) ToDocument(ext *entity.PluginExtension) *document.PluginExtensionDocument {
	if ext == nil {
		return nil
	}

	doc := &document.PluginExtensionDocument{
		NumericID: ext.ID,
		PluginID:  ext.PluginID,
		Name:      ext.Name,
		Type:      string(ext.Type),
		Path:      ext.Path,
		Handler:   ext.Handler,
		Config:    ext.Config,
		CreatedAt: ext.CreatedAt,
	}

	if ext.DeletedAt.Valid {
		doc.DeletedAt = &ext.DeletedAt.Time
	}

	return doc
}

// ToEntity converts a PluginExtensionDocument to a PluginExtension entity.
// Note: The Plugin relationship is not populated - use a separate query if needed.
func (m *PluginExtensionMapper) ToEntity(doc *document.PluginExtensionDocument) *entity.PluginExtension {
	if doc == nil {
		return nil
	}

	ext := &entity.PluginExtension{
		ID:        doc.NumericID,
		PluginID:  doc.PluginID,
		Name:      doc.Name,
		Type:      entity.PluginType(doc.Type),
		Path:      doc.Path,
		Handler:   doc.Handler,
		Config:    doc.Config,
		CreatedAt: doc.CreatedAt,
	}

	if doc.DeletedAt != nil {
		ext.DeletedAt = gorm.DeletedAt{Time: *doc.DeletedAt, Valid: true}
	}

	return ext
}

// ToEntities converts a slice of PluginExtensionDocument to a slice of PluginExtension entities.
func (m *PluginExtensionMapper) ToEntities(docs []*document.PluginExtensionDocument) []*entity.PluginExtension {
	if docs == nil {
		return nil
	}

	extensions := make([]*entity.PluginExtension, len(docs))
	for i, doc := range docs {
		extensions[i] = m.ToEntity(doc)
	}
	return extensions
}

// ToDocuments converts a slice of PluginExtension entities to a slice of PluginExtensionDocument.
func (m *PluginExtensionMapper) ToDocuments(extensions []*entity.PluginExtension) []*document.PluginExtensionDocument {
	if extensions == nil {
		return nil
	}

	docs := make([]*document.PluginExtensionDocument, len(extensions))
	for i, ext := range extensions {
		docs[i] = m.ToDocument(ext)
	}
	return docs
}
