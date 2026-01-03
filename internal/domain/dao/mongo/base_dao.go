// Package mongo provides MongoDB-based DAO implementations.
package mongo

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// IDCounter manages auto-incrementing IDs for MongoDB documents.
// This provides SQL-like uint IDs for compatibility with the domain entities.
type IDCounter struct {
	collection *mongo.Collection
	mu         sync.Mutex
}

// counterDocument represents the structure stored in the counters collection.
type counterDocument struct {
	ID    string `bson:"_id"`
	Value uint   `bson:"value"`
}

// NewIDCounter creates a new IDCounter for a MongoDB database.
func NewIDCounter(db *mongo.Database) *IDCounter {
	return &IDCounter{
		collection: db.Collection("counters"),
	}
}

// NextID returns the next available ID for a given collection.
func (c *IDCounter) NextID(ctx context.Context, collectionName string) (uint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filter := bson.M{"_id": collectionName}
	update := bson.M{"$inc": bson.M{"value": 1}}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var counter counterDocument
	err := c.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&counter)
	if err != nil {
		return 0, err
	}

	return counter.Value, nil
}

// baseMongoDAO provides common MongoDB operations for all entity DAOs.
type baseMongoDAO[T any, D any] struct {
	collection *mongo.Collection
	idCounter  *IDCounter
}

// newBaseMongoDAO creates a new base MongoDB DAO instance.
func newBaseMongoDAO[T any, D any](db *mongo.Database, collectionName string, idCounter *IDCounter) *baseMongoDAO[T, D] {
	return &baseMongoDAO[T, D]{
		collection: db.Collection(collectionName),
		idCounter:  idCounter,
	}
}

// getCollection returns the MongoDB collection.
func (d *baseMongoDAO[T, D]) getCollection() *mongo.Collection {
	return d.collection
}

// getIDCounter returns the ID counter.
func (d *baseMongoDAO[T, D]) getIDCounter() *IDCounter {
	return d.idCounter
}

// nextID generates the next available ID for this collection.
func (d *baseMongoDAO[T, D]) nextID(ctx context.Context) (uint, error) {
	return d.idCounter.NextID(ctx, d.collection.Name())
}

// notDeletedFilter returns a filter that excludes soft-deleted documents.
func notDeletedFilter() bson.M {
	return bson.M{"deleted_at": nil}
}

// withNotDeleted adds the not-deleted condition to an existing filter.
func withNotDeleted(filter bson.M) bson.M {
	filter["deleted_at"] = nil
	return filter
}

// count returns the count of documents matching the filter.
func (d *baseMongoDAO[T, D]) count(ctx context.Context, filter bson.M) (int64, error) {
	return d.collection.CountDocuments(ctx, filter)
}

// existsBy checks if a document exists by a field value.
func (d *baseMongoDAO[T, D]) existsBy(ctx context.Context, field string, value any) (bool, error) {
	filter := withNotDeleted(bson.M{field: value})
	count, err := d.collection.CountDocuments(ctx, filter)
	return count > 0, err
}

// findOneByFilter finds a single document matching the filter.
func (d *baseMongoDAO[T, D]) findOneByFilter(ctx context.Context, filter bson.M, result any) error {
	return d.collection.FindOne(ctx, filter).Decode(result)
}

// findManyByFilter finds all documents matching the filter.
func (d *baseMongoDAO[T, D]) findManyByFilter(ctx context.Context, filter bson.M, opts *options.FindOptions, results any) error {
	cursor, err := d.collection.Find(ctx, filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	return cursor.All(ctx, results)
}

// insertOne inserts a single document.
func (d *baseMongoDAO[T, D]) insertOne(ctx context.Context, doc any) error {
	_, err := d.collection.InsertOne(ctx, doc)
	return err
}

// updateOne updates a single document matching the filter.
func (d *baseMongoDAO[T, D]) updateOne(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := d.collection.UpdateOne(ctx, filter, update)
	return err
}

// updateMany updates all documents matching the filter.
func (d *baseMongoDAO[T, D]) updateMany(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := d.collection.UpdateMany(ctx, filter, update)
	return err
}

// deleteMany deletes all documents matching the filter.
func (d *baseMongoDAO[T, D]) deleteMany(ctx context.Context, filter bson.M) error {
	_, err := d.collection.DeleteMany(ctx, filter)
	return err
}
