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

// userDAO implements dao.UserDAO using MongoDB.
type userDAO struct {
	*baseMongoDAO[entity.User, document.UserDocument]
	mapper *mapper.UserMapper
}

// NewUserDAO creates a new MongoDB-based UserDAO.
func NewUserDAO(db *mongo.Database, idCounter *IDCounter) dao.UserDAO {
	return &userDAO{
		baseMongoDAO: newBaseMongoDAO[entity.User, document.UserDocument](
			db,
			document.UserDocument{}.CollectionName(),
			idCounter,
		),
		mapper: mapper.NewUserMapper(),
	}
}

// Create inserts a new user into MongoDB.
func (d *userDAO) Create(ctx context.Context, user *entity.User) error {
	// Generate numeric ID for compatibility
	id, err := d.nextID(ctx)
	if err != nil {
		return err
	}
	user.ID = id
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	doc := d.mapper.ToDocument(user)
	return d.insertOne(ctx, doc)
}

// FindByID retrieves a user by their numeric ID.
func (d *userDAO) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	filter := withNotDeleted(bson.M{"numeric_id": id})

	var doc document.UserDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.mapper.ToEntity(&doc), nil
}

// Update modifies an existing user in MongoDB.
func (d *userDAO) Update(ctx context.Context, user *entity.User) error {
	user.UpdatedAt = time.Now()
	doc := d.mapper.ToDocument(user)

	filter := bson.M{"numeric_id": user.ID}
	update := bson.M{"$set": doc}
	return d.updateOne(ctx, filter, update)
}

// Delete performs a soft delete on a user.
func (d *userDAO) Delete(ctx context.Context, id uint) error {
	now := time.Now()
	filter := bson.M{"numeric_id": id}
	update := bson.M{"$set": bson.M{"deleted_at": now}}
	return d.updateOne(ctx, filter, update)
}

// FindAll retrieves users with pagination.
func (d *userDAO) FindAll(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
	filter := notDeletedFilter()

	total, err := d.count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * size)
	opts := options.Find().
		SetSkip(skip).
		SetLimit(int64(size)).
		SetSort(bson.D{{Key: "numeric_id", Value: -1}})

	var docs []*document.UserDocument
	if err := d.findManyByFilter(ctx, filter, opts, &docs); err != nil {
		return nil, 0, err
	}

	return d.mapper.ToEntities(docs), total, nil
}

// Count returns the total number of users.
func (d *userDAO) Count(ctx context.Context) (int64, error) {
	return d.count(ctx, notDeletedFilter())
}

// ExistsBy checks if a user exists by a field value.
func (d *userDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	return d.existsBy(ctx, field, value)
}

// FindByUsername retrieves a user by their username.
func (d *userDAO) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	filter := withNotDeleted(bson.M{"username": username})

	var doc document.UserDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.mapper.ToEntity(&doc), nil
}

// FindByEmail retrieves a user by their email.
func (d *userDAO) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	filter := withNotDeleted(bson.M{"email": email})

	var doc document.UserDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.mapper.ToEntity(&doc), nil
}

// FindByUsernameOrEmail retrieves a user by username or email.
func (d *userDAO) FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"username": usernameOrEmail},
			{"email": usernameOrEmail},
		},
		"deleted_at": nil,
	}

	var doc document.UserDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.mapper.ToEntity(&doc), nil
}

// ExistsByUsername checks if a user with the given username exists.
func (d *userDAO) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return d.existsBy(ctx, "username", username)
}

// ExistsByEmail checks if a user with the given email exists.
func (d *userDAO) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return d.existsBy(ctx, "email", email)
}
