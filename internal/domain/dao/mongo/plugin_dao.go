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

// pluginDAO implements dao.PluginDAO using MongoDB.
type pluginDAO struct {
	*baseMongoDAO[entity.Plugin, document.PluginDocument]
	mapper *mapper.PluginMapper
}

// NewPluginDAO creates a new MongoDB-based PluginDAO.
func NewPluginDAO(db *mongo.Database, idCounter *IDCounter) dao.PluginDAO {
	return &pluginDAO{
		baseMongoDAO: newBaseMongoDAO[entity.Plugin, document.PluginDocument](
			db,
			document.PluginDocument{}.CollectionName(),
			idCounter,
		),
		mapper: mapper.NewPluginMapper(),
	}
}

// Create inserts a new plugin into MongoDB.
func (d *pluginDAO) Create(ctx context.Context, plugin *entity.Plugin) error {
	// Generate numeric ID for compatibility
	id, err := d.nextID(ctx)
	if err != nil {
		return err
	}
	plugin.ID = id
	plugin.CreatedAt = time.Now()
	plugin.UpdatedAt = time.Now()

	doc := d.mapper.ToDocument(plugin)
	return d.insertOne(ctx, doc)
}

// FindByID retrieves a plugin by its numeric ID.
func (d *pluginDAO) FindByID(ctx context.Context, id uint) (*entity.Plugin, error) {
	filter := withNotDeleted(bson.M{"numeric_id": id})

	var doc document.PluginDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.mapper.ToEntity(&doc), nil
}

// Update modifies an existing plugin in MongoDB.
func (d *pluginDAO) Update(ctx context.Context, plugin *entity.Plugin) error {
	plugin.UpdatedAt = time.Now()
	doc := d.mapper.ToDocument(plugin)

	filter := bson.M{"numeric_id": plugin.ID}
	update := bson.M{"$set": doc}
	return d.updateOne(ctx, filter, update)
}

// Delete performs a soft delete on a plugin.
func (d *pluginDAO) Delete(ctx context.Context, id uint) error {
	now := time.Now()
	filter := bson.M{"numeric_id": id}
	update := bson.M{"$set": bson.M{"deleted_at": now, "updated_at": now}}
	return d.updateOne(ctx, filter, update)
}

// FindAll retrieves plugins with pagination.
func (d *pluginDAO) FindAll(ctx context.Context, page, size int) ([]*entity.Plugin, int64, error) {
	filter := notDeletedFilter()

	total, err := d.count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * size)
	opts := options.Find().
		SetSkip(skip).
		SetLimit(int64(size)).
		SetSort(bson.D{{Key: "installed_at", Value: -1}})

	var docs []*document.PluginDocument
	if err := d.findManyByFilter(ctx, filter, opts, &docs); err != nil {
		return nil, 0, err
	}

	return d.mapper.ToEntities(docs), total, nil
}

// Count returns the total number of plugins.
func (d *pluginDAO) Count(ctx context.Context) (int64, error) {
	return d.count(ctx, notDeletedFilter())
}

// ExistsBy checks if a plugin exists by a field value.
func (d *pluginDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	return d.existsBy(ctx, field, value)
}

// FindByKey retrieves a plugin by its unique key.
func (d *pluginDAO) FindByKey(ctx context.Context, key string) (*entity.Plugin, error) {
	filter := withNotDeleted(bson.M{"key": key})

	var doc document.PluginDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.mapper.ToEntity(&doc), nil
}

// DeleteByKey soft-deletes a plugin by its key.
func (d *pluginDAO) DeleteByKey(ctx context.Context, key string) error {
	now := time.Now()
	filter := bson.M{"key": key}
	update := bson.M{"$set": bson.M{"deleted_at": now, "updated_at": now}}
	return d.updateOne(ctx, filter, update)
}

// FindByState retrieves all plugins with a specific state.
func (d *pluginDAO) FindByState(ctx context.Context, state entity.PluginState) ([]*entity.Plugin, error) {
	filter := withNotDeleted(bson.M{"state": string(state)})

	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	var docs []*document.PluginDocument
	if err := d.findManyByFilter(ctx, filter, opts, &docs); err != nil {
		return nil, err
	}

	return d.mapper.ToEntities(docs), nil
}

// FindEnabled retrieves all plugins that are currently enabled.
func (d *pluginDAO) FindEnabled(ctx context.Context) ([]*entity.Plugin, error) {
	return d.FindByState(ctx, entity.PluginStateEnabled)
}

// ExistsByKey checks if a plugin with the given key exists.
func (d *pluginDAO) ExistsByKey(ctx context.Context, key string) (bool, error) {
	return d.existsBy(ctx, "key", key)
}

// UpdateState updates the state of a plugin by its ID.
func (d *pluginDAO) UpdateState(ctx context.Context, id uint, state entity.PluginState) error {
	now := time.Now()
	updates := bson.M{
		"state":      string(state),
		"updated_at": now,
	}

	// Set enabled_at when enabling the plugin
	if state == entity.PluginStateEnabled {
		updates["enabled_at"] = now
	}

	filter := bson.M{"numeric_id": id}
	update := bson.M{"$set": updates}
	return d.updateOne(ctx, filter, update)
}
