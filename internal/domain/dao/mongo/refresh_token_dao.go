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

// refreshTokenDAO implements dao.RefreshTokenDAO using MongoDB.
type refreshTokenDAO struct {
	*baseMongoDAO[entity.RefreshToken, document.RefreshTokenDocument]
	mapper  *mapper.RefreshTokenMapper
	userDAO dao.UserDAO
}

// NewRefreshTokenDAO creates a new MongoDB-based RefreshTokenDAO.
func NewRefreshTokenDAO(db *mongo.Database, idCounter *IDCounter, userDAO dao.UserDAO) dao.RefreshTokenDAO {
	return &refreshTokenDAO{
		baseMongoDAO: newBaseMongoDAO[entity.RefreshToken, document.RefreshTokenDocument](
			db,
			document.RefreshTokenDocument{}.CollectionName(),
			idCounter,
		),
		mapper:  mapper.NewRefreshTokenMapper(),
		userDAO: userDAO,
	}
}

// Create inserts a new refresh token into MongoDB.
func (d *refreshTokenDAO) Create(ctx context.Context, token *entity.RefreshToken) error {
	// Generate numeric ID for compatibility
	id, err := d.nextID(ctx)
	if err != nil {
		return err
	}
	token.ID = id
	token.CreatedAt = time.Now()

	doc := d.mapper.ToDocument(token)
	return d.insertOne(ctx, doc)
}

// FindByID retrieves a refresh token by its numeric ID.
func (d *refreshTokenDAO) FindByID(ctx context.Context, id uint) (*entity.RefreshToken, error) {
	filter := withNotDeleted(bson.M{"numeric_id": id})

	var doc document.RefreshTokenDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	token := d.mapper.ToEntity(&doc)

	// Load the associated user
	if d.userDAO != nil {
		user, err := d.userDAO.FindByID(ctx, token.UserID)
		if err == nil && user != nil {
			token.User = *user
		}
	}

	return token, nil
}

// Update modifies an existing refresh token in MongoDB.
func (d *refreshTokenDAO) Update(ctx context.Context, token *entity.RefreshToken) error {
	doc := d.mapper.ToDocument(token)

	filter := bson.M{"numeric_id": token.ID}
	update := bson.M{"$set": doc}
	return d.updateOne(ctx, filter, update)
}

// Delete performs a soft delete on a refresh token.
func (d *refreshTokenDAO) Delete(ctx context.Context, id uint) error {
	now := time.Now()
	filter := bson.M{"numeric_id": id}
	update := bson.M{"$set": bson.M{"deleted_at": now}}
	return d.updateOne(ctx, filter, update)
}

// FindAll retrieves refresh tokens with pagination.
func (d *refreshTokenDAO) FindAll(ctx context.Context, page, size int) ([]*entity.RefreshToken, int64, error) {
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

	var docs []*document.RefreshTokenDocument
	if err := d.findManyByFilter(ctx, filter, opts, &docs); err != nil {
		return nil, 0, err
	}

	return d.mapper.ToEntities(docs), total, nil
}

// Count returns the total number of refresh tokens.
func (d *refreshTokenDAO) Count(ctx context.Context) (int64, error) {
	return d.count(ctx, notDeletedFilter())
}

// ExistsBy checks if a refresh token exists by a field value.
func (d *refreshTokenDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	return d.existsBy(ctx, field, value)
}

// FindByToken retrieves a refresh token by its value.
// Only returns non-revoked tokens.
func (d *refreshTokenDAO) FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	filter := bson.M{
		"token":      token,
		"revoked":    false,
		"deleted_at": nil,
	}

	var doc document.RefreshTokenDocument
	err := d.findOneByFilter(ctx, filter, &doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	refreshToken := d.mapper.ToEntity(&doc)

	// Load the associated user
	if d.userDAO != nil {
		user, err := d.userDAO.FindByID(ctx, refreshToken.UserID)
		if err == nil && user != nil {
			refreshToken.User = *user
		}
	}

	return refreshToken, nil
}

// RevokeByToken revokes a specific refresh token.
func (d *refreshTokenDAO) RevokeByToken(ctx context.Context, token string) error {
	filter := bson.M{"token": token}
	update := bson.M{"$set": bson.M{"revoked": true}}
	return d.updateOne(ctx, filter, update)
}

// RevokeAllByUserID revokes all refresh tokens for a specific user.
func (d *refreshTokenDAO) RevokeAllByUserID(ctx context.Context, userID uint) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"revoked": true}}
	return d.updateMany(ctx, filter, update)
}

// DeleteExpired removes all expired tokens from the database.
func (d *refreshTokenDAO) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	filter := bson.M{"expires_at": bson.M{"$lt": now}}
	return d.deleteMany(ctx, filter)
}
