package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao/mongo/document"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao/mongo/mapper"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// pluginExtensionDAO implements dao.PluginExtensionDAO using MongoDB.
type pluginExtensionDAO struct {
	*baseMongoDAO[entity.PluginExtension, document.PluginExtensionDocument]
	mapper *mapper.PluginExtensionMapper
}

// NewPluginExtensionDAO creates a new MongoDB-based PluginExtensionDAO.
func NewPluginExtensionDAO(db *mongo.Database, idCounter *IDCounter) dao.PluginExtensionDAO {
	return &pluginExtensionDAO{
		baseMongoDAO: newBaseMongoDAO[entity.PluginExtension, document.PluginExtensionDocument](
			db,
			document.PluginExtensionDocument{}.CollectionName(),
			idCounter,
		),
		mapper: mapper.NewPluginExtensionMapper(),
	}
}

// Create inserts a new plugin extension into MongoDB.
func (d *pluginExtensionDAO) Create(ctx context.Context, ext *entity.PluginExtension) error {
	// Generate numeric ID for compatibility
	id, err := d.nextID(ctx)
	if err != nil {
		return err
	}
	ext.ID = id
	ext.CreatedAt = time.Now()

	doc := d.mapper.ToDocument(ext)
	return d.insertOne(ctx, doc)
}

// FindByID retrieves a plugin extension by its numeric ID.
func (d *pluginExtensionDAO) FindByID(ctx context.Context, id uint) (*entity.PluginExtension, error) {
	filter := withNotDeleted(bson.M{"numeric_id": id})

	var doc document.PluginExtensionDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.mapper.ToEntity(&doc), nil
}

// Update modifies an existing plugin extension in MongoDB.
func (d *pluginExtensionDAO) Update(ctx context.Context, ext *entity.PluginExtension) error {
	doc := d.mapper.ToDocument(ext)

	filter := bson.M{"numeric_id": ext.ID}
	update := bson.M{"$set": doc}
	return d.updateOne(ctx, filter, update)
}

// Delete performs a soft delete on a plugin extension.
func (d *pluginExtensionDAO) Delete(ctx context.Context, id uint) error {
	now := time.Now()
	filter := bson.M{"numeric_id": id}
	update := bson.M{"$set": bson.M{"deleted_at": now}}
	return d.updateOne(ctx, filter, update)
}

// FindAll retrieves plugin extensions with pagination.
func (d *pluginExtensionDAO) FindAll(ctx context.Context, page, size int) ([]*entity.PluginExtension, int64, error) {
	filter := notDeletedFilter()

	total, err := d.count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * size)
	opts := options.Find().
		SetSkip(skip).
		SetLimit(int64(size)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	var docs []*document.PluginExtensionDocument
	if err := d.findManyByFilter(ctx, filter, opts, &docs); err != nil {
		return nil, 0, err
	}

	return d.mapper.ToEntities(docs), total, nil
}

// Count returns the total number of plugin extensions.
func (d *pluginExtensionDAO) Count(ctx context.Context) (int64, error) {
	return d.count(ctx, notDeletedFilter())
}

// ExistsBy checks if a plugin extension exists by a field value.
func (d *pluginExtensionDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	return d.existsBy(ctx, field, value)
}

// FindByPluginID retrieves all extensions belonging to a specific plugin.
func (d *pluginExtensionDAO) FindByPluginID(ctx context.Context, pluginID uint) ([]*entity.PluginExtension, error) {
	filter := withNotDeleted(bson.M{"plugin_id": pluginID})

	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	var docs []*document.PluginExtensionDocument
	if err := d.findManyByFilter(ctx, filter, opts, &docs); err != nil {
		return nil, err
	}

	return d.mapper.ToEntities(docs), nil
}

// DeleteByPluginID deletes all extensions belonging to a specific plugin.
func (d *pluginExtensionDAO) DeleteByPluginID(ctx context.Context, pluginID uint) error {
	now := time.Now()
	filter := bson.M{"plugin_id": pluginID}
	update := bson.M{"$set": bson.M{"deleted_at": now}}
	return d.updateMany(ctx, filter, update)
}
