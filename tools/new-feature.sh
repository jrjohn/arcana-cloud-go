#!/usr/bin/env bash
# Generate one vertical CRUD slice in this template's layers (AGENTS.md §1-§2):
#
#   tools/new-feature.sh Invoice
#
#   entity        internal/domain/entity/<x>.go
#   DAO           internal/domain/dao/<x>_dao.go           + impl/gorm/<x>_dao_impl.go (+ test)
#   repository    internal/domain/repository/<x>_repository.go + impl/<x>_repository_impl.go (+ test)
#   DTOs          internal/dto/request/<x>.go, internal/dto/response/<x>.go
#   service       internal/domain/service/<x>_service.go    + impl/<x>_service_impl.go (+ test)
#   controller    internal/controller/http/<x>_controller.go (+ test)
#   wiring        internal/di/{dao,repository,service,controller,server,database}.go
#
# Mirrors the Plugin/User features. Every wiring edit needs an anchor line in internal/di/;
# if one is missing the script stops with an error instead of leaving the feature unwired.
# MongoDB: only the GORM (MySQL/PostgreSQL) DAO is generated; the DI provider returns an
# explicit error under ARCANA_DATABASE_DRIVER=mongodb until you add impl/mongo/<x>_dao_impl.go
# (mirror plugin_dao_impl.go + document/ + mapper/).
set -euo pipefail

[ $# -eq 1 ] || { echo "usage: $0 <PascalCaseName>   e.g. $0 Invoice" >&2; exit 1; }
X="$1"
[[ "$X" =~ ^[A-Z][A-Za-z0-9]*$ ]] || { echo "name must be PascalCase (e.g. Invoice, PurchaseOrder)" >&2; exit 1; }
command -v perl >/dev/null || { echo "perl is required" >&2; exit 1; }

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
MOD="$(awk '/^module /{print $2; exit}' go.mod)"
[ -n "$MOD" ] || { echo "new-feature: no module line in go.mod" >&2; exit 1; }
LOWER="$(printf '%s' "${X:0:1}" | tr 'A-Z' 'a-z')${X:1}"
SNAKE="$(printf '%s' "$X" | perl -pe 's/([a-z0-9])([A-Z])/$1_$2/g; $_=lc')"
ROUTE="$(printf '%s' "$SNAKE" | tr '_' '-')s"

for f in internal/domain/entity/$SNAKE.go internal/domain/dao/${SNAKE}_dao.go internal/controller/http/${SNAKE}_controller.go; do
    [ ! -e "$f" ] || { echo "new-feature: $f already exists — refusing to overwrite" >&2; exit 1; }
done
if grep -q "provide${X}DAO\|${X}Controller" internal/di/*.go; then
    echo "new-feature: ${X} is already wired in internal/di/" >&2; exit 1
fi

# ---- 1. check every anchor BEFORE writing anything, so a failure leaves the tree untouched
need() { # file, perl regex (multi-line)
    perl -0ne "exit(/$2/ ? 0 : 1)" "$1" || { echo "new-feature: anchor /$2/ not found in $1 — wiring would be skipped; fix the anchor in tools/new-feature.sh" >&2; exit 1; }
}
need internal/di/dao.go 'var DAOModule = fx\.Module\("dao",\n\tfx\.Provide\(\n'
need internal/di/repository.go 'var RepositoryModule = fx\.Module\("repository",\n\tfx\.Provide\(\n'
need internal/di/service.go 'var ServiceModule = fx\.Module\("service",\n\tfx\.Provide\(\n'
need internal/di/controller.go 'var ControllerModule = fx\.Module\("controller",\n\tfx\.Provide\(\n'
need internal/di/server.go 'type Controllers struct \{\n\tfx\.In\n\n'
need internal/di/server.go 'api := router\.Group\("\/api\/v1"\)\n'
need internal/di/database.go 'return sqlDB\.DB\.AutoMigrate\(\n'
need internal/di/dao.go 'import \(\n'

render() { # stdin template -> file
    mkdir -p "$(dirname "$1")"
    perl -pe "s/__X__/${X}/g; s/__x__/${LOWER}/g; s/__snake__/${SNAKE}/g; s/__route__/${ROUTE}/g; s#__MOD__#${MOD}#g" > "$1"
    CREATED+=("$1")
}
CREATED=()

# ---- 2. entity
render internal/domain/entity/$SNAKE.go <<'EOF'
package entity

import (
	"time"

	"gorm.io/gorm"
)

// __X__ is the __x__ entity. Persistence shape only; APIs expose response.__X__Response.
type __X__ struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:200;not null" json:"name"`
	Description string         `gorm:"size:1000" json:"description,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
EOF

# ---- 3. DAO interface + GORM impl + test
render internal/domain/dao/${SNAKE}_dao.go <<'EOF'
package dao

import (
	"__MOD__/internal/domain/entity"
)

// __X__DAO is the data access interface for __X__ (one data source per implementation).
type __X__DAO interface {
	BaseDAO[entity.__X__, uint]
}
EOF

render internal/domain/dao/impl/gorm/${SNAKE}_dao_impl.go <<'EOF'
package gorm

import (
	"gorm.io/gorm"

	"__MOD__/internal/domain/dao"
	"__MOD__/internal/domain/entity"
)

// __x__DAO implements dao.__X__DAO using GORM for SQL databases.
type __x__DAO struct {
	*baseGormDAO[entity.__X__]
}

// New__X__DAO creates a new GORM-based __X__DAO.
func New__X__DAO(db *gorm.DB) dao.__X__DAO {
	return &__x__DAO{baseGormDAO: newBaseGormDAO[entity.__X__](db)}
}
EOF

render internal/domain/dao/impl/gorm/${SNAKE}_dao_test.go <<'EOF'
package gorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"__MOD__/internal/domain/entity"
)

func setup__X__TestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.__X__{}))
	return db
}

func Test__X__DAO_CRUD(t *testing.T) {
	d := New__X__DAO(setup__X__TestDB(t))
	ctx := context.Background()

	item := &entity.__X__{Name: "first", Description: "desc"}
	require.NoError(t, d.Create(ctx, item))
	assert.NotZero(t, item.ID)

	found, err := d.FindByID(ctx, item.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "first", found.Name)

	found.Name = "renamed"
	require.NoError(t, d.Update(ctx, found))

	items, total, err := d.FindAll(ctx, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "renamed", items[0].Name)

	require.NoError(t, d.Delete(ctx, item.ID))
	gone, err := d.FindByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Nil(t, gone)
}
EOF

# ---- 4. repository interface + impl + test
render internal/domain/repository/${SNAKE}_repository.go <<'EOF'
package repository

import (
	"context"

	"__MOD__/internal/domain/entity"
)

// __X__Repository defines domain-level data operations for __X__.
type __X__Repository interface {
	Create(ctx context.Context, item *entity.__X__) error
	GetByID(ctx context.Context, id uint) (*entity.__X__, error)
	Update(ctx context.Context, item *entity.__X__) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, page, size int) ([]*entity.__X__, int64, error)
}
EOF

render internal/domain/repository/impl/${SNAKE}_repository_impl.go <<'EOF'
package impl

import (
	"context"

	"__MOD__/internal/domain/dao"
	"__MOD__/internal/domain/entity"
	"__MOD__/internal/domain/repository"
)

// __x__Repository implements repository.__X__Repository by delegating to __X__DAO.
type __x__Repository struct {
	dao dao.__X__DAO
}

// New__X__Repository creates a new __X__Repository instance.
func New__X__Repository(__x__DAO dao.__X__DAO) repository.__X__Repository {
	return &__x__Repository{dao: __x__DAO}
}

// Create inserts a new __x__.
func (r *__x__Repository) Create(ctx context.Context, item *entity.__X__) error {
	return r.dao.Create(ctx, item)
}

// GetByID retrieves a __x__ by ID (nil, nil when absent).
func (r *__x__Repository) GetByID(ctx context.Context, id uint) (*entity.__X__, error) {
	return r.dao.FindByID(ctx, id)
}

// Update modifies an existing __x__.
func (r *__x__Repository) Update(ctx context.Context, item *entity.__X__) error {
	return r.dao.Update(ctx, item)
}

// Delete removes a __x__ by ID.
func (r *__x__Repository) Delete(ctx context.Context, id uint) error {
	return r.dao.Delete(ctx, id)
}

// List retrieves __x__ items with pagination.
func (r *__x__Repository) List(ctx context.Context, page, size int) ([]*entity.__X__, int64, error) {
	return r.dao.FindAll(ctx, page, size)
}
EOF

render internal/domain/repository/impl/${SNAKE}_repository_impl_test.go <<'EOF'
package impl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"__MOD__/internal/domain/entity"
)

type mock__X__DAO struct{ mock.Mock }

func (m *mock__X__DAO) Create(ctx context.Context, e *entity.__X__) error {
	return m.Called(ctx, e).Error(0)
}
func (m *mock__X__DAO) FindByID(ctx context.Context, id uint) (*entity.__X__, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.__X__), args.Error(1)
}
func (m *mock__X__DAO) Update(ctx context.Context, e *entity.__X__) error {
	return m.Called(ctx, e).Error(0)
}
func (m *mock__X__DAO) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mock__X__DAO) FindAll(ctx context.Context, page, size int) ([]*entity.__X__, int64, error) {
	args := m.Called(ctx, page, size)
	return args.Get(0).([]*entity.__X__), args.Get(1).(int64), args.Error(2)
}
func (m *mock__X__DAO) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mock__X__DAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	args := m.Called(ctx, field, value)
	return args.Bool(0), args.Error(1)
}

func Test__X__Repository_Delegates(t *testing.T) {
	ctx := context.Background()
	d := new(mock__X__DAO)
	repo := New__X__Repository(d)
	item := &entity.__X__{ID: 7, Name: "a"}

	d.On("Create", ctx, item).Return(nil)
	d.On("FindByID", ctx, uint(7)).Return(item, nil)
	d.On("Update", ctx, item).Return(nil)
	d.On("Delete", ctx, uint(7)).Return(nil)
	d.On("FindAll", ctx, 1, 10).Return([]*entity.__X__{item}, int64(1), nil)

	assert.NoError(t, repo.Create(ctx, item))
	got, err := repo.GetByID(ctx, 7)
	assert.NoError(t, err)
	assert.Equal(t, item, got)
	assert.NoError(t, repo.Update(ctx, item))
	assert.NoError(t, repo.Delete(ctx, 7))
	items, total, err := repo.List(ctx, 1, 10)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, int64(1), total)
	d.AssertExpectations(t)
}
EOF

# ---- 5. DTOs
render internal/dto/request/${SNAKE}.go <<'EOF'
package request

// Create__X__Request is the body of POST /api/v1/__route__.
type Create__X__Request struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=1000"`
}

// Update__X__Request is the body of PUT /api/v1/__route__/:id.
type Update__X__Request struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=1000"`
}
EOF

render internal/dto/response/${SNAKE}.go <<'EOF'
package response

import "time"

// __X__Response is the API shape of a __x__ (never return entity.__X__ directly).
type __X__Response struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
EOF

# ---- 6. service interface + impl + test
render internal/domain/service/${SNAKE}_service.go <<'EOF'
package service

import (
	"context"
	"errors"

	"__MOD__/internal/dto/request"
	"__MOD__/internal/dto/response"
)

// Err__X__NotFound is returned when a __x__ does not exist.
var Err__X__NotFound = errors.New("__snake__ not found")

// __X__Service defines the business operations for __X__.
type __X__Service interface {
	Create(ctx context.Context, req *request.Create__X__Request) (*response.__X__Response, error)
	GetByID(ctx context.Context, id uint) (*response.__X__Response, error)
	List(ctx context.Context, page, size int) (*response.PagedResponse[response.__X__Response], error)
	Update(ctx context.Context, id uint, req *request.Update__X__Request) (*response.__X__Response, error)
	Delete(ctx context.Context, id uint) error
}
EOF

render internal/domain/service/impl/${SNAKE}_service_impl.go <<'EOF'
package impl

import (
	"context"

	"__MOD__/internal/domain/entity"
	"__MOD__/internal/domain/repository"
	"__MOD__/internal/domain/service"
	"__MOD__/internal/dto/request"
	"__MOD__/internal/dto/response"
)

// __x__Service implements service.__X__Service.
type __x__Service struct {
	repo repository.__X__Repository
}

// New__X__Service creates a new __X__Service instance.
func New__X__Service(repo repository.__X__Repository) service.__X__Service {
	return &__x__Service{repo: repo}
}

// Create validates and stores a new __x__.
func (s *__x__Service) Create(ctx context.Context, req *request.Create__X__Request) (*response.__X__Response, error) {
	item := &entity.__X__{Name: req.Name, Description: req.Description}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return s.to__X__Response(item), nil
}

// GetByID returns one __x__ or service.Err__X__NotFound.
func (s *__x__Service) GetByID(ctx context.Context, id uint) (*response.__X__Response, error) {
	item, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.to__X__Response(item), nil
}

// List returns one page of __x__ items; page and size are clamped to sane values.
func (s *__x__Service) List(ctx context.Context, page, size int) (*response.PagedResponse[response.__X__Response], error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	items, total, err := s.repo.List(ctx, page, size)
	if err != nil {
		return nil, err
	}
	out := make([]response.__X__Response, len(items))
	for i, item := range items {
		out[i] = *s.to__X__Response(item)
	}
	result := response.NewPagedResponse(out, page, size, total)
	return &result, nil
}

// Update changes an existing __x__.
func (s *__x__Service) Update(ctx context.Context, id uint, req *request.Update__X__Request) (*response.__X__Response, error) {
	item, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	item.Name = req.Name
	item.Description = req.Description
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.to__X__Response(item), nil
}

// Delete removes a __x__ (service.Err__X__NotFound when absent).
func (s *__x__Service) Delete(ctx context.Context, id uint) error {
	if _, err := s.find(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *__x__Service) find(ctx context.Context, id uint) (*entity.__X__, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, service.Err__X__NotFound
	}
	return item, nil
}

func (s *__x__Service) to__X__Response(item *entity.__X__) *response.__X__Response {
	return &response.__X__Response{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}
EOF

render internal/domain/service/impl/${SNAKE}_service_impl_test.go <<'EOF'
package impl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"__MOD__/internal/domain/entity"
	"__MOD__/internal/domain/service"
	"__MOD__/internal/dto/request"
)

// fake__X__Repository is an in-memory repository.__X__Repository (no database).
type fake__X__Repository struct {
	items  map[uint]*entity.__X__
	nextID uint
}

func newFake__X__Repository() *fake__X__Repository {
	return &fake__X__Repository{items: map[uint]*entity.__X__{}}
}

func (r *fake__X__Repository) Create(_ context.Context, e *entity.__X__) error {
	r.nextID++
	e.ID = r.nextID
	r.items[e.ID] = e
	return nil
}
func (r *fake__X__Repository) GetByID(_ context.Context, id uint) (*entity.__X__, error) {
	return r.items[id], nil
}
func (r *fake__X__Repository) Update(_ context.Context, e *entity.__X__) error {
	r.items[e.ID] = e
	return nil
}
func (r *fake__X__Repository) Delete(_ context.Context, id uint) error {
	delete(r.items, id)
	return nil
}
func (r *fake__X__Repository) List(_ context.Context, _, _ int) ([]*entity.__X__, int64, error) {
	out := make([]*entity.__X__, 0, len(r.items))
	for _, e := range r.items {
		out = append(out, e)
	}
	return out, int64(len(out)), nil
}

func Test__X__Service_CRUD(t *testing.T) {
	ctx := context.Background()
	svc := New__X__Service(newFake__X__Repository())

	created, err := svc.Create(ctx, &request.Create__X__Request{Name: "a", Description: "d"})
	require.NoError(t, err)
	assert.Equal(t, "a", created.Name)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	updated, err := svc.Update(ctx, created.ID, &request.Update__X__Request{Name: "b"})
	require.NoError(t, err)
	assert.Equal(t, "b", updated.Name)

	page, err := svc.List(ctx, 0, 0)
	require.NoError(t, err)
	assert.Len(t, page.Items, 1)

	require.NoError(t, svc.Delete(ctx, created.ID))
	_, err = svc.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, service.Err__X__NotFound)
	assert.ErrorIs(t, svc.Delete(ctx, created.ID), service.Err__X__NotFound)
	_, err = svc.Update(ctx, created.ID, &request.Update__X__Request{Name: "c"})
	assert.ErrorIs(t, err, service.Err__X__NotFound)
}
EOF

# ---- 7. controller + test
render internal/controller/http/${SNAKE}_controller.go <<'EOF'
package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"__MOD__/internal/domain/service"
	"__MOD__/internal/dto/request"
	"__MOD__/internal/dto/response"
	"__MOD__/internal/middleware"
)

// __X__Controller exposes /api/v1/__route__. Handlers only extract, delegate and wrap.
type __X__Controller struct {
	__x__Service   service.__X__Service
	authMiddleware *middleware.AuthMiddleware
}

// New__X__Controller creates a new __X__Controller.
func New__X__Controller(svc service.__X__Service, authMiddleware *middleware.AuthMiddleware) *__X__Controller {
	return &__X__Controller{__x__Service: svc, authMiddleware: authMiddleware}
}

// RegisterRoutes registers the __x__ routes (all authenticated).
func (c *__X__Controller) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/__route__")
	g.Use(c.authMiddleware.Authenticate())
	g.GET("", c.List)
	g.GET("/:id", c.GetByID)
	g.POST("", c.Create)
	g.PUT("/:id", c.Update)
	g.DELETE("/:id", c.Delete)
}

// List godoc: GET /api/v1/__route__?page=&size=
func (c *__X__Controller) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	result, err := c.__x__Service.List(ctx.Request.Context(), page, size)
	if err != nil {
		write__X__Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewSuccessWithData(result))
}

// GetByID godoc: GET /api/v1/__route__/:id
func (c *__X__Controller) GetByID(ctx *gin.Context) {
	id, ok := parse__X__ID(ctx)
	if !ok {
		return
	}
	result, err := c.__x__Service.GetByID(ctx.Request.Context(), id)
	if err != nil {
		write__X__Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewSuccessWithData(result))
}

// Create godoc: POST /api/v1/__route__
func (c *__X__Controller) Create(ctx *gin.Context) {
	var req request.Create__X__Request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any]("invalid request", err.Error()))
		return
	}
	result, err := c.__x__Service.Create(ctx.Request.Context(), &req)
	if err != nil {
		write__X__Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, response.NewSuccess(result, "__snake__ created"))
}

// Update godoc: PUT /api/v1/__route__/:id
func (c *__X__Controller) Update(ctx *gin.Context) {
	id, ok := parse__X__ID(ctx)
	if !ok {
		return
	}
	var req request.Update__X__Request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any]("invalid request", err.Error()))
		return
	}
	result, err := c.__x__Service.Update(ctx.Request.Context(), id, &req)
	if err != nil {
		write__X__Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewSuccess(result, "__snake__ updated"))
}

// Delete godoc: DELETE /api/v1/__route__/:id
func (c *__X__Controller) Delete(ctx *gin.Context) {
	id, ok := parse__X__ID(ctx)
	if !ok {
		return
	}
	if err := c.__x__Service.Delete(ctx.Request.Context(), id); err != nil {
		write__X__Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "__snake__ deleted"))
}

func parse__X__ID(ctx *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("invalid id"))
		return 0, false
	}
	return uint(id), true
}

func write__X__Error(ctx *gin.Context, err error) {
	if errors.Is(err, service.Err__X__NotFound) {
		ctx.JSON(http.StatusNotFound, response.NewError[any](err.Error()))
		return
	}
	ctx.JSON(http.StatusInternalServerError, response.NewError[any]("internal error"))
}
EOF

render internal/controller/http/${SNAKE}_controller_test.go <<'EOF'
package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"__MOD__/internal/domain/service"
	"__MOD__/internal/dto/request"
	"__MOD__/internal/dto/response"
)

// stub__X__Service is a hand-written service.__X__Service fake (no database).
type stub__X__Service struct{ err error }

func (s *stub__X__Service) Create(_ context.Context, req *request.Create__X__Request) (*response.__X__Response, error) {
	return &response.__X__Response{ID: 1, Name: req.Name}, s.err
}
func (s *stub__X__Service) GetByID(_ context.Context, id uint) (*response.__X__Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &response.__X__Response{ID: id}, nil
}
func (s *stub__X__Service) List(_ context.Context, page, size int) (*response.PagedResponse[response.__X__Response], error) {
	r := response.NewPagedResponse([]response.__X__Response{{ID: 1}}, page, size, 1)
	return &r, s.err
}
func (s *stub__X__Service) Update(_ context.Context, id uint, req *request.Update__X__Request) (*response.__X__Response, error) {
	return &response.__X__Response{ID: id, Name: req.Name}, s.err
}
func (s *stub__X__Service) Delete(_ context.Context, _ uint) error { return s.err }

func new__X__TestRouter(t *testing.T, svc service.__X__Service) *gin.Engine {
	securityService, jwtProvider := setupSecurityService(t)
	c := New__X__Controller(svc, setupAuthMiddleware(t, jwtProvider, securityService))
	r := setupTestRouter()
	r.GET("/x", c.List)
	r.GET("/x/:id", c.GetByID)
	r.POST("/x", c.Create)
	r.PUT("/x/:id", c.Update)
	r.DELETE("/x/:id", c.Delete)
	c.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func do__X__(r *gin.Engine, method, path, body string) int {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w.Code
}

func Test__X__Controller_Handlers(t *testing.T) {
	r := new__X__TestRouter(t, &stub__X__Service{})
	cases := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, "/x", "", http.StatusOK},
		{http.MethodGet, "/x/3", "", http.StatusOK},
		{http.MethodGet, "/x/abc", "", http.StatusBadRequest},
		{http.MethodPost, "/x", `{"name":"a"}`, http.StatusCreated},
		{http.MethodPost, "/x", `{}`, http.StatusBadRequest},
		{http.MethodPut, "/x/3", `{"name":"b"}`, http.StatusOK},
		{http.MethodDelete, "/x/3", "", http.StatusOK},
		{http.MethodGet, "/api/v1/__route__", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		if got := do__X__(r, tc.method, tc.path, tc.body); got != tc.want {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.path, got, tc.want)
		}
	}
}

func Test__X__Controller_NotFound(t *testing.T) {
	r := new__X__TestRouter(t, &stub__X__Service{err: service.Err__X__NotFound})
	if got := do__X__(r, http.MethodGet, "/x/9", ""); got != http.StatusNotFound {
		t.Errorf("GET /x/9 = %d, want 404", got)
	}
	if got := do__X__(r, http.MethodDelete, "/x/9", ""); got != http.StatusNotFound {
		t.Errorf("DELETE /x/9 = %d, want 404", got)
	}
}
EOF

# ---- 8. wiring (anchors were verified in step 1)
ins() { # file, anchor regex, text to insert after anchor
    perl -0pi -e "s/($2)/\$1$3/ or die" "$1"
}
ins internal/di/dao.go 'import \(\n' '\t"fmt"\n'
ins internal/di/dao.go 'var DAOModule = fx\.Module\("dao",\n\tfx\.Provide\(\n' "\\t\\tprovide${X}DAO,\\n"
cat >> internal/di/dao.go <<EOT

// provide${X}DAO creates a ${X}DAO. Only the GORM (MySQL/PostgreSQL) implementation exists;
// under MongoDB this fails at startup instead of running without the feature.
func provide${X}DAO(cfg *config.DatabaseConfig, sqlDB *SQLDatabase) (dao.${X}DAO, error) {
	if cfg.IsMongoDB() {
		return nil, fmt.Errorf("${X}DAO: no MongoDB implementation yet (add internal/domain/dao/impl/mongo/${SNAKE}_dao_impl.go)")
	}
	return gormdao.New${X}DAO(sqlDB.DB), nil
}
EOT
ins internal/di/repository.go 'var RepositoryModule = fx\.Module\("repository",\n\tfx\.Provide\(\n' "\\t\\tprovide${X}Repository,\\n"
cat >> internal/di/repository.go <<EOT

// provide${X}Repository creates a ${X}Repository that delegates to ${X}DAO.
func provide${X}Repository(d dao.${X}DAO) repository.${X}Repository {
	return impl.New${X}Repository(d)
}
EOT
ins internal/di/service.go 'var ServiceModule = fx\.Module\("service",\n\tfx\.Provide\(\n' "\\t\\tprovide${X}Service,\\n"
cat >> internal/di/service.go <<EOT

func provide${X}Service(repo repository.${X}Repository) service.${X}Service {
	return serviceimpl.New${X}Service(repo)
}
EOT
ins internal/di/controller.go 'var ControllerModule = fx\.Module\("controller",\n\tfx\.Provide\(\n' "\\t\\tprovide${X}Controller,\\n"
cat >> internal/di/controller.go <<EOT

func provide${X}Controller(
	svc service.${X}Service,
	authMiddleware *middleware.AuthMiddleware,
) *httpctrl.${X}Controller {
	return httpctrl.New${X}Controller(svc, authMiddleware)
}
EOT
ins internal/di/server.go 'type Controllers struct \{\n\tfx\.In\n\n' "\\t${X} *httpctrl.${X}Controller\\n"
ins internal/di/server.go 'api := router\.Group\("\/api\/v1"\)\n' "\\n\\tcontrollers.${X}.RegisterRoutes(api)\\n"
ins internal/di/database.go 'return sqlDB\.DB\.AutoMigrate\(\n' "\\t\\t\\t\\&entity.${X}{},\\n"

gofmt -w internal/di/{dao,repository,service,controller,server,database}.go "${CREATED[@]}" 2>/dev/null || echo "note: gofmt not found — run gofmt -w internal/ before committing" >&2

echo "created ${X} slice:"; printf '  %s\n' "${CREATED[@]}"
echo "wired:   internal/di/{dao,repository,service,controller,server,database}.go"
echo "next:    go build ./... && go test ./internal/... && tools/arch-check.sh"
